package models

import (
	"context"
	"crynux_relay/config"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
)

type ChainTaskStatus uint8

const (
	ChainTaskQueued ChainTaskStatus = iota
	ChainTaskStarted
	ChainTaskParametersUploaded
	ChainTaskErrorReported
	ChainTaskScoreReady
	ChainTaskValidated
	ChainTaskGroupValidated
	ChainTaskEndInvalidated
	ChainTaskEndSuccess
	ChainTaskEndAborted
	ChainTaskEndGroupRefund
	ChainTaskEndGroupSuccess
)

type TaskStatus uint8

const (
	InferenceTaskCreated TaskStatus = iota
	InferenceTaskParamsUploaded
	InferenceTaskResultsReady
	InferenceTaskEndAborted
	InferenceTaskEndGroupRefund
	InferenceTaskEndInvalidated
	InferenceTaskEndSuccess
)

type ChainTaskType uint8

const (
	TaskTypeSD ChainTaskType = iota
	TaskTypeLLM
	TaskTypeSDFTLora
)

type TaskAbortReason uint8

const (
	TaskAbortReasonNone TaskAbortReason = iota
	TaskAbortTimeout
	TaskAbortModelDownloadFailed
	TaskAbortIncorrectResult
	TaskAbortTaskFeeTooLow
)

type TaskError uint8

const (
	TaskErrorNone TaskError = iota
	TaskErrorParametersValidationFailed
)

type InferenceTask struct {
	gorm.Model
	TaskArgs         string          `json:"task_args"`
	TaskIDCommitment string          `json:"task_id_commitment" gorm:"index"`
	Creator          string          `json:"creator"`
	Status           TaskStatus      `json:"status"`
	TaskType         ChainTaskType   `json:"task_type" gorm:"index"`
	MinVRAM          uint64          `json:"min_vram"`
	RequiredGPU      string          `json:"required_gpu"`
	RequiredGPUVRAM  uint64          `json:"required_gpu_vram"`
	TaskFee          float64         `json:"task_fee"`
	TaskSize         uint64          `json:"task_size"`
	ModelID          string          `json:"model_id"`
	AbortReason      TaskAbortReason `json:"abort_reason"`
	TaskError        TaskError       `json:"task_error"`
	SelectedNode     string          `json:"selected_node"`
	// time when task is created (get from blockchain)
	CreateTime sql.NullTime `json:"create_time" gorm:"index;null;default:null"`
	// time when relay report task params are uploaded
	StartTime sql.NullTime `json:"start_time" gorm:"index;null;default:null"`
	// time when task score is ready (get from blockchain)
	ScoreReadyTime sql.NullTime `json:"score_ready_time" gorm:"index;null;default:null"`
	// time when relay find that task score is validated
	ValidatedTime sql.NullTime `json:"validated_time" gorm:"index;null;default:null"`
	// time when relay report task results are uploaded
	ResultUploadedTime sql.NullTime `json:"result_uploaded_time" gorm:"index;null;default:null"`
}

func (task *InferenceTask) Save(ctx context.Context) error {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := config.GetDB().WithContext(dbCtx).Save(&task).Error; err != nil {
		return err
	}
	return nil
}

func (task *InferenceTask) Update(ctx context.Context, newTask *InferenceTask) error {
	if task.ID == 0 {
		return errors.New("InferenceTask.ID cannot be 0 when update")
	}
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := config.GetDB().WithContext(dbCtx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(task).Updates(newTask).Error; err != nil {
			return err
		}
		if err := tx.Model(task).First(task).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func GetTaskByIDCommitment(ctx context.Context, taskIDCommitment string) (*InferenceTask, error) {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	task := InferenceTask{TaskIDCommitment: taskIDCommitment}
	if err := config.GetDB().WithContext(dbCtx).Model(&task).Where(&task).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}
