package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	log "github.com/sirupsen/logrus"
)

func generateQueuedTasks(ctx context.Context, taskCh chan<- *models.InferenceTask) error {
	startID := uint(0)
	limit := 100

	for {
		tasks, err := func(ctx context.Context, startID uint, limit int) ([]models.InferenceTask, error) {
			tasks := make([]models.InferenceTask, 0)

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
		for _, task := range tasks {
			taskCh <- &task
		}
		startID = tasks[len(tasks)-1].ID
	}
}

func processQueuedTask(ctx context.Context, taskCh chan *models.InferenceTask) error {
	for task := range taskCh {
		selectedNode, err := selectNodeForInferenceTask(ctx, task)
		if err != nil {
			return err
		}
		if selectedNode == nil {
			taskCh <- task
		}
		SetTaskStatusStarted(ctx, config.GetDB(), task, selectedNode)
	}
	return nil
}

func StartTaskProcesser(ctx context.Context) {
	taskCh := make(chan *models.InferenceTask, 100)

	go func(ctx context.Context, taskCh chan<- *models.InferenceTask) {
		for {
			err := generateQueuedTasks(ctx, taskCh)
			if err == context.DeadlineExceeded || err == context.Canceled {
				close(taskCh)
				return
			}
			if err != nil {
				log.Errorf("StartTask: generate queued tasks error: %v", err)
			}
			time.Sleep(2 * time.Second)
		}
	}(ctx, taskCh)

	for {
		err := processQueuedTask(ctx, taskCh)
		if err == context.DeadlineExceeded || err == context.Canceled {
			return
		}
		if err != nil {
			log.Errorf("StartTask: process queued tasks error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}
}
