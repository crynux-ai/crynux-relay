package service

import (
	"container/heap"
	"crynux_relay/models"
	"sync"
)

type TaskWithRetry struct {
	Task       *models.InferenceTask
	RetryCount int
}

type taskPriorityQueue []*TaskWithRetry

func (pq taskPriorityQueue) Len() int { return len(pq) }

func (pq taskPriorityQueue) Less(i, j int) bool {
	taskFeeI := &pq[i].Task.TaskFee.Int
	taskFeeJ := &pq[j].Task.TaskFee.Int

	flag := taskFeeI.Cmp(taskFeeJ)
	if flag > 0 {
		return true
	} else if flag < 0 {
		return false
	}
	if pq[i].RetryCount < pq[j].RetryCount {
		return true
	} else if pq[i].RetryCount > pq[j].RetryCount {
		return false
	}
	if pq[i].Task.TaskType > pq[j].Task.TaskType {
		return true
	} else if pq[i].Task.TaskType < pq[j].Task.TaskType {
		return false
	}
	if pq[i].Task.ID < pq[j].Task.ID {
		return true
	}
	return false
}

func (pq taskPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *taskPriorityQueue) Push(x any) {
	item := x.(*TaskWithRetry)
	*pq = append(*pq, item)
}

func (pq *taskPriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[0 : n-1]
	return item
}

type TaskQueue struct {
	cond   *sync.Cond
	queue  taskPriorityQueue
	closed bool
}

func NewTaskQueue() *TaskQueue {
	l := sync.Mutex{}
	cond := sync.NewCond(&l)
	return &TaskQueue{
		cond:   cond,
		queue:  make(taskPriorityQueue, 0),
		closed: false,
	}
}

func (q *TaskQueue) Push(task ...*models.InferenceTask) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	for _, t := range task {
		heap.Push(&q.queue, &TaskWithRetry{
			Task:       t,
			RetryCount: 0,
		})
	}
	q.cond.Broadcast()
}

func (q *TaskQueue) PushWithRetry(task *models.InferenceTask, retryCount int) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	heap.Push(&q.queue, &TaskWithRetry{
		Task:       task,
		RetryCount: retryCount,
	})
	q.cond.Broadcast()
}

func (q *TaskQueue) Pop() (*models.InferenceTask, int) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	for q.queue.Len() == 0 && !q.closed {
		q.cond.Wait()
	}
	if q.closed {
		return nil, 0
	}
	taskWithRetry := heap.Pop(&q.queue).(*TaskWithRetry)
	return taskWithRetry.Task, taskWithRetry.RetryCount
}

func (q *TaskQueue) Close() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.closed = true
	q.cond.Broadcast()
}
