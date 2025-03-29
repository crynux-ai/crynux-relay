package service

import (
	"container/heap"
	"crynux_relay/models"
	"sync"
)

type taskPriorityQueue []*models.InferenceTask

func (pq taskPriorityQueue) Len() int { return len(pq) }

func (pq taskPriorityQueue) Less(i, j int) bool {
	flag := pq[i].TaskFee.Cmp(&pq[j].TaskFee.Int)
	if flag > 0 {
		return true
	} else if flag == 0 {
		return pq[i].TaskType > pq[j].TaskType
	}
	return false
}

func (pq taskPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *taskPriorityQueue) Push(x any) {
	item := x.(*models.InferenceTask)
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
	for _, task := range task {
		heap.Push(&q.queue, task)
	}
	q.cond.Broadcast()
}

func (q *TaskQueue) Pop() *models.InferenceTask {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	for q.queue.Len() == 0 && !q.closed {
		q.cond.Wait()
	}
	if q.closed {
		return nil
	}
	return heap.Pop(&q.queue).(*models.InferenceTask)
}

func (q *TaskQueue) Close() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.closed = true
	q.cond.Broadcast()
}
