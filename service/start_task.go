package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"
	"errors"
	"math/rand"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type DispatchedTask struct {
	task      *models.InferenceTask
	node      *models.Node
	resChan   chan bool
	createdAt time.Time
	finished  bool
	mu        sync.RWMutex
}

type TaskDispatcher struct {
	nodeQueue        chan string
	taskMap          sync.Map
	processingTasks  sync.Map
	dispatchLimiter  chan struct{}
	startTaskLimiter chan struct{}
}

func NewTaskDispatcher() *TaskDispatcher {
	return &TaskDispatcher{
		nodeQueue:        make(chan string, 100),
		dispatchLimiter:  make(chan struct{}, 100),
		startTaskLimiter: make(chan struct{}, 100),
	}
}

func (d *TaskDispatcher) getQueuedTasks(ctx context.Context) {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	limit := 100
	for {
		select {
		case <-ctx.Done():
			return
		default:
			tasks, err := func(ctx context.Context) ([]*models.InferenceTask, error) {
				tasks := make([]*models.InferenceTask, 0)

				dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
					Where("status = ?", models.TaskQueued).
					Order("id").
					Limit(limit).
					Find(&tasks).Error
				if err != nil {
					return nil, err
				}
				return tasks, nil
			}(ctx)
			if err == nil && len(tasks) > 0 {
				for _, task := range tasks {
					if _, loaded := d.processingTasks.LoadOrStore(task.ID, struct{}{}); loaded {
						continue
					}
					go func(task *models.InferenceTask) {
						d.dispatchLimiter <- struct{}{}
						defer func() {
							<-d.dispatchLimiter
							d.processingTasks.Delete(task.ID)
						}()

						if err := task.Sync(ctx, config.GetDB()); err != nil {
							log.Errorf("StartTask: sync task status error: %v", err)
							return
						}
						if task.Status != models.TaskQueued {
							return
						}

						deadline := task.CreateTime.Time.Add(3*time.Minute + time.Duration(task.Timeout)*time.Second)
						if deadline.Before(time.Now()) {
							log.Debugf("StartTask: task %s timeout, abort", task.TaskIDCommitment)
							task.AbortReason = models.TaskAbortTimeout
							task.ValidatedTime = sql.NullTime{Time: time.Now(), Valid: true}
							ctx1, cancel := context.WithTimeout(ctx, 10*time.Second)
							defer cancel()
							appConfig := config.GetConfig()
							if err := SetTaskStatusEndAborted(ctx1, config.GetDB(), task, appConfig.Blockchain.Account.Address); err != nil {
								log.Errorf("StartTask: abort task %s error: %v", task.TaskIDCommitment, err)
							}
							return
						}
						ctx1, cancel := context.WithDeadline(ctx, deadline)
						defer cancel()
						log.Debugf("StartTask: dispatch task %s", task.TaskIDCommitment)
						d.Dispatch(ctx1, task)
					}(task)

				}
			} else {
				if err != nil {
					log.Errorf("StartTask: get queued tasks error: %v", err)

				}

				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(2 * time.Second)

				select {
				case <-ctx.Done():
					return
				case <-timer.C:
				}
			}
		}
	}
}

func (d *TaskDispatcher) Process(ctx context.Context, task *models.InferenceTask, node *models.Node) bool {
	dispatchedTask, loaded := d.taskMap.LoadOrStore(node.Address, &DispatchedTask{
		task:      task,
		node:      node,
		resChan:   make(chan bool, 1),
		createdAt: time.Now(),
		finished:  false,
	})
	if !loaded {
		log.Debugf("StartTask: new dispatched task %s on node %s", task.TaskIDCommitment, node.Address)
		resChan := dispatchedTask.(*DispatchedTask).resChan
		d.nodeQueue <- node.Address
		log.Debugf("StartTask: waiting for task %s on node %s", task.TaskIDCommitment, node.Address)
		select {
		case res := <-resChan:
			return res
		case <-ctx.Done():
			return false
		}

	} else {
		dispatchedTask, _ := dispatchedTask.(*DispatchedTask)
		if dispatchedTask.mu.TryLock() {
			if dispatchedTask.finished {
				dispatchedTask.mu.Unlock()
				log.Debugf("StartTask: node %s has been dispatched a task, skip", node.Address)
				return false
			}
			originalTask := dispatchedTask.task
			if originalTask.TaskFee.Cmp(&task.TaskFee.Int) >= 0 {
				dispatchedTask.mu.Unlock()
				log.Debugf("StartTask: task %s fee is lower than original task fee, skip", task.TaskIDCommitment)
				return false
			}
			// if current task fee is higher than original task fee, replace the original task
			log.Debugf("StartTask: task %s fee is higher than original task fee, replace", task.TaskIDCommitment)
			log.Debugf("StartTask: task %s is replaced by task %s", originalTask.TaskIDCommitment, task.TaskIDCommitment)
			dispatchedTask.task = task
			dispatchedTask.resChan <- false
			newResChan := make(chan bool, 1)
			dispatchedTask.resChan = newResChan
			dispatchedTask.mu.Unlock()
			log.Debugf("StartTask: waiting for task %s on node %s", task.TaskIDCommitment, node.Address)
			select {
			case res := <-newResChan:
				return res
			case <-ctx.Done():
				return false
			}
		}
		log.Debugf("StartTask: node %s is dispatching", node.Address)
		return false
	}

}

func (d *TaskDispatcher) Dispatch(ctx context.Context, task *models.InferenceTask) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			selectedNode, err := selectNodeForInferenceTask(ctx, task)

			if err == nil && selectedNode != nil {
				log.Debugf("StartTask: select node %s for task: %s", selectedNode.Address, task.TaskIDCommitment)
				ok := d.Process(ctx, task, selectedNode)
				if ok {
					log.Debugf("StartTask: dispatch task %s to node %s success", task.TaskIDCommitment, selectedNode.Address)
					return
				} else {
					log.Debugf("StartTask: dispatch task %s to node %s failed", task.TaskIDCommitment, selectedNode.Address)
				}
			} else if err != nil {
				log.Errorf("StartTask: select node for task %s error: %v", task.TaskIDCommitment, err)
			} else if selectedNode == nil {
				log.Debugf("StartTask: no available node for task %s", task.TaskIDCommitment)
			}
			randomSleep := rand.Intn(500) + 500
			time.Sleep(time.Duration(randomSleep) * time.Millisecond)
		}
	}
}

func (d *TaskDispatcher) processDispatchedTasks(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case nodeAddress := <-d.nodeQueue:
			t, exists := d.taskMap.Load(nodeAddress)
			if !exists {
				log.Debugf("StartTask: node %s is not dispatching any task, skip", nodeAddress)
				continue
			}
			dispatchedTask, _ := t.(*DispatchedTask)
			log.Debugf("StartTask: start processing dispatched tasks, task %s started on node %s", dispatchedTask.task.TaskIDCommitment, dispatchedTask.node.Address)

			if time.Now().Before(dispatchedTask.createdAt.Add(time.Second)) {
				log.Debugf("StartTask: task %s is still waiting for other tasks, skip", dispatchedTask.task.TaskIDCommitment)
				d.nodeQueue <- nodeAddress
			} else {
				go func() {
					d.startTaskLimiter <- struct{}{}
					defer func() {
						<-d.startTaskLimiter
					}()

					dispatchedTask.mu.Lock()
					err := SetTaskStatusStarted(ctx, config.GetDB(), dispatchedTask.task, dispatchedTask.node)
					success := err == nil

					dispatchedTask.resChan <- success
					dispatchedTask.finished = true
					dispatchedTask.mu.Unlock()

					d.taskMap.Delete(dispatchedTask.node.Address)

					if success {
						log.Debugf("StartTask: process dispatched tasks success, task %s started on node %s", dispatchedTask.task.TaskIDCommitment, dispatchedTask.node.Address)
					} else {
						if errors.Is(err, errWrongTaskStatus) || errors.Is(err, models.ErrTaskStatusChanged) {
							log.Debugf("StartTask: process dispatched tasks failed, task %s status changed", dispatchedTask.task.TaskIDCommitment)
						} else if errors.Is(err, models.ErrNodeStatusChanged) {
							log.Debugf("StartTask: process dispatched tasks failed, node %s status changed", dispatchedTask.node.Address)
						} else {
							log.Errorf("StartTask: process dispatched tasks error: %v", err)
						}
					}
				}()
			}

		}
	}
}

func StartTaskProcesser(ctx context.Context) {
	taskDispatcher := NewTaskDispatcher()

	go taskDispatcher.processDispatchedTasks(ctx)
	go taskDispatcher.getQueuedTasks(ctx)
}
