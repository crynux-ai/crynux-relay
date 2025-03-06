package models

import (
	"context"
	"crynux_relay/config"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

var TaskVersionInvalidError = errors.New("Task version invalid")

type TaskStatus uint8

const (
	TaskQueued TaskStatus = iota
	TaskStarted
	TaskParametersUploaded
	TaskErrorReported
	TaskScoreReady
	TaskValidated
	TaskGroupValidated
	TaskEndInvalidated
	TaskEndSuccess
	TaskEndAborted
	TaskEndGroupRefund
	TaskEndGroupSuccess
)

type TaskType uint8

const (
	TaskTypeSD TaskType = iota
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

type StringArray []string

func (arr *StringArray) Scan(val interface{}) error {
	var arrString string
	switch v := val.(type) {
	case string:
		arrString = v
	case []byte:
		arrString = string(v)
	case nil:
		return nil
	default:
		return errors.New(fmt.Sprint("Unable to parse value to StringArray: ", val))
	}
	*arr = strings.Split(arrString, ";")
	return nil
}

func (arr StringArray) Value() (driver.Value, error) {
	res := strings.Join(arr, ";")
	return res, nil
}

func (arr StringArray) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(arr))
}

func (arr *StringArray) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, (*[]string)(arr))
}

type InferenceTask struct {
	gorm.Model
	TaskArgs         string          `json:"task_args"`
	TaskIDCommitment string          `json:"task_id_commitment" gorm:"index"`
	Creator          string          `json:"creator"`
	SamplingSeed     string          `json:"sampling_seed"`
	Nonce            string          `json:"nonce"`
	Status           TaskStatus      `json:"status"`
	TaskType         TaskType        `json:"task_type" gorm:"index"`
	TaskVersion      string          `json:"task_version"`
	Timeout          uint64          `json:"timeout"`
	MinVRAM          uint64          `json:"min_vram"`
	RequiredGPU      string          `json:"required_gpu"`
	RequiredGPUVRAM  uint64          `json:"required_gpu_vram"`
	TaskFee          float64         `json:"task_fee"`
	TaskSize         uint64          `json:"task_size"`
	ModelIDs         StringArray     `json:"model_ids" gorm:"type:text"`
	AbortReason      TaskAbortReason `json:"abort_reason"`
	TaskError        TaskError       `json:"task_error"`
	SelectedNode     string          `json:"selected_node"`
	// time when task is created (get from blockchain)
	CreateTime sql.NullTime `json:"create_time" gorm:"index;null;default:null"`
	// time when task is started (get from blockchain)
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
