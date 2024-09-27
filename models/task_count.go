package models

import (
	"time"

	"gorm.io/gorm"
)

type TaskCount struct {
	gorm.Model

	Start        time.Time     `json:"start"`
	End          time.Time     `json:"end"`
	TaskType     ChainTaskType `json:"task_type"`
	TotalCount   int64         `json:"total_count"`
	SuccessCount int64         `json:"success_count"`
	AbortedCount int64         `json:"aborted_count"`
}

type TaskExecutionTimeCount struct {
	gorm.Model

	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	TaskType ChainTaskType `json:"task_type"`
	Seconds  int64         `json:"seconds"`
	Count    int64         `json:"count"`
}