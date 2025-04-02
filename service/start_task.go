package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
)

func generateQueuedTasks(ctx context.Context, taskQueue *TaskQueue) error {
	startID := uint(0)
	limit := 100

	for {
		tasks, err := func(ctx context.Context, startID uint, limit int) ([]*models.InferenceTask, error) {
			tasks := make([]*models.InferenceTask, 0)

			dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
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
		taskQueue.Push(tasks...)
		startID = tasks[len(tasks)-1].ID
	}
}

func processQueuedTask(ctx context.Context, taskQueue *TaskQueue) error {
	for {
		task := taskQueue.Pop()
		if task == nil {
			break
		}
		selectedNode, err := selectNodeForInferenceTask(ctx, task)
		if err != nil {
			if err != context.DeadlineExceeded && err != context.Canceled {
				taskQueue.Push(task)
			}
			return err
		}
		if selectedNode == nil {
			go func(task *models.InferenceTask) {
				time.Sleep(2 * time.Second)
				taskQueue.Push(task)
			}(task)
		} else {
			err := SetTaskStatusStarted(ctx, config.GetDB(), task, selectedNode)
			if err != nil && !errors.Is(err, errWrongTaskStatus) && !errors.Is(err, models.ErrTaskStatusChanged) {
				taskQueue.Push(task)
			}
		}
	}
	return nil
}

func StartTaskProcesser(ctx context.Context) {
	taskQueue := NewTaskQueue()

	go func(ctx context.Context, taskQueue *TaskQueue) {
		for {
			err := generateQueuedTasks(ctx, taskQueue)
			if err == context.DeadlineExceeded || err == context.Canceled {
				taskQueue.Close()
				return
			}
			if err != nil {
				log.Errorf("StartTask: generate queued tasks error: %v", err)
			}
			time.Sleep(2 * time.Second)
		}
	}(ctx, taskQueue)

	for {
		err := processQueuedTask(ctx, taskQueue)
		if err == context.DeadlineExceeded || err == context.Canceled {
			break
		}
		if err != nil {
			log.Errorf("StartTask: process queued tasks error: %v", err)
		}
		time.Sleep(2 * time.Second)
	}
}
