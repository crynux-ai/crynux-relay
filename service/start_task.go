package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"
	"math/rand"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var startID uint = 0

func generateQueuedTasks(ctx context.Context, taskQueue chan<- *models.InferenceTask) error {
	limit := 100

	for {
		tasks, err := func(ctx context.Context, startID uint, limit int) ([]*models.InferenceTask, error) {
			tasks := make([]*models.InferenceTask, 0)

			dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
				Where("status = ?", models.TaskQueued).
				Where("id > ?", startID).
				Order("id").
				Limit(limit).
				Find(&tasks).Error
			if err != nil {
				return nil, err
			}
			return tasks, nil
		}(ctx, startID, limit)
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			time.Sleep(2 * time.Second)
			continue
		}
		for _, task := range tasks {
			taskQueue <- task
		}
		startID = tasks[len(tasks)-1].ID
	}
}

type DispatchedTask struct {
	task    *models.InferenceTask
	node    *models.Node
	resChan chan bool
	mu      sync.RWMutex
}

type TaskDispatcher struct {
	nodeQueue []string
	taskMap   map[string]*DispatchedTask
	mu        sync.RWMutex
}

func NewTaskDispatcher() *TaskDispatcher {
	return &TaskDispatcher{
		nodeQueue: make([]string, 0),
		taskMap:   make(map[string]*DispatchedTask),
	}
}

func (d *TaskDispatcher) Process(ctx context.Context, task *models.InferenceTask, node *models.Node) bool {
	d.mu.Lock()
	dispatchedTask, exists := d.taskMap[node.Address]
	if !exists {
		log.Infof("StartTask: new dispatched task %s on node %s", task.TaskIDCommitment, node.Address)
		d.nodeQueue = append(d.nodeQueue, node.Address)
		resChan := make(chan bool, 1)
		d.taskMap[node.Address] = &DispatchedTask{
			task:    task,
			node:    node,
			resChan: resChan,
		}
		d.mu.Unlock()
		log.Infof("StartTask: waiting for task %s on node %s", task.TaskIDCommitment, node.Address)
		select {
		case res := <-resChan:
			return res
		case <-ctx.Done():
			return false
		}

	} else {
		if dispatchedTask.mu.TryLock() {
			originalTask := dispatchedTask.task
			if originalTask.TaskFee.Cmp(&task.TaskFee.Int) >= 0 {
				dispatchedTask.mu.Unlock()
				d.mu.Unlock()
				log.Infof("StartTask: task %s fee is lower than original task fee, skip", task.TaskIDCommitment)
				return false
			}
			// if current task fee is higher than original task fee, replace the original task
			log.Infof("StartTask: task %s fee is higher than original task fee, replace", task.TaskIDCommitment)
			log.Infof("StartTask: task %s is replaced by task %s", originalTask.TaskIDCommitment, task.TaskIDCommitment)
			dispatchedTask.task = task
			dispatchedTask.resChan <- false
			newResChan := make(chan bool, 1)
			dispatchedTask.resChan = newResChan
			dispatchedTask.mu.Unlock()
			d.mu.Unlock()
			log.Infof("StartTask: waiting for task %s on node %s", task.TaskIDCommitment, node.Address)
			select {
			case res := <-newResChan:
				return res
			case <-ctx.Done():
				return false
			}
		}
		d.mu.Unlock()
		log.Infof("StartTask: node %s is dispatching", node.Address)
		return false
	}

}

func (d *TaskDispatcher) Dispatch(ctx context.Context, task *models.InferenceTask) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := task.SyncStatus(ctx, config.GetDB()); err != nil {
				log.Errorf("StartTask: sync task status error: %v", err)
				continue
			}
			if task.Status != models.TaskQueued {
				return
			}

			selectedNode, err := selectNodeForInferenceTask(ctx, task)
			
			if err == nil && selectedNode != nil {
				log.Infof("StartTask: select node %s for task: %s", selectedNode.Address, task.TaskIDCommitment)
				ok := d.Process(ctx, task, selectedNode)
				if ok {
					log.Infof("StartTask: dispatch task %s to node %s success", task.TaskIDCommitment, selectedNode.Address)
					return
				} else {
					log.Infof("StartTask: dispatch task %s to node %s failed", task.TaskIDCommitment, selectedNode.Address)
				}
			}
			if err != nil {
				log.Errorf("StartTask: select node for task %s error: %v", task.TaskIDCommitment, err)
			} 
			if selectedNode == nil {
				log.Errorf("StartTask: no available node for task %s", task.TaskIDCommitment)
			}
			randomSleep := rand.Intn(1000) + 1000
			time.Sleep(time.Duration(randomSleep) * time.Millisecond)
		}
	}
}

func (d *TaskDispatcher) ProcessDispatchedTasks(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			d.mu.RLock()
			if len(d.nodeQueue) == 0 {
				d.mu.RUnlock()
				time.Sleep(1 * time.Second)
				log.Infoln("StartTask: no dispatched tasks")
				continue
			}
			nodeAddress := d.nodeQueue[0]
			dispatchedTask := d.taskMap[nodeAddress]
			log.Infof("StartTask: start processing dispatched tasks, task %s started on node %s", dispatchedTask.task.TaskIDCommitment, dispatchedTask.node.Address)
			d.mu.RUnlock()

			dispatchedTask.mu.Lock()
			err := SetTaskStatusStarted(ctx, config.GetDB(), dispatchedTask.task, dispatchedTask.node)
			if err == nil {
				log.Infof("StartTask: process dispatched tasks success, task %s started on node %s", dispatchedTask.task.TaskIDCommitment, dispatchedTask.node.Address)
				d.mu.Lock()
				delete(d.taskMap, dispatchedTask.node.Address)
				d.nodeQueue = d.nodeQueue[1:]
				d.mu.Unlock()
				dispatchedTask.resChan <- true
				dispatchedTask.mu.Unlock()
			} else if errors.Is(err, errWrongTaskStatus) || errors.Is(err, models.ErrTaskStatusChanged) {
				log.Infof("StartTask: process dispatched tasks failed, task %s status changed", dispatchedTask.task.TaskIDCommitment)
				d.mu.Lock()
				delete(d.taskMap, dispatchedTask.node.Address)
				d.nodeQueue = d.nodeQueue[1:]
				d.mu.Unlock()
				dispatchedTask.mu.Unlock()
				dispatchedTask.resChan <- false
			} else {
				log.Errorf("StartTask: process dispatched tasks error: %v", err)
				dispatchedTask.mu.Unlock()
			}
		}
	}
}

func StartTaskProcesser(ctx context.Context) {
	taskQueue := make(chan *models.InferenceTask)
	taskDispatcher := NewTaskDispatcher()

	go taskDispatcher.ProcessDispatchedTasks(ctx)

	go func(ctx context.Context, taskQueue chan *models.InferenceTask) {
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()

		for {
			err := generateQueuedTasks(ctx, taskQueue)
			if err != nil {
				log.Errorf("StartTask: generate queued tasks error: %v", err)
			}

			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(2 * time.Second)

			select {
			case <-ctx.Done():
				close(taskQueue)
				return
			case <-timer.C:
			}
		}
	}(ctx, taskQueue)

	for {
		select {
		case <-ctx.Done():
			return
		case task := <-taskQueue:
			go func(task *models.InferenceTask) {
				deadline := task.CreateTime.Time.Add(3 * time.Minute)
				if deadline.Before(time.Now()) {
					return
				}
				ctx1, cancel := context.WithDeadline(ctx, deadline)
				defer cancel()
				log.Infof("StartTask: dispatch task %s", task.TaskIDCommitment)
				taskDispatcher.Dispatch(ctx1, task)
			}(task)
		}
	}
}
