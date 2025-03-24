package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"gorm.io/gorm"
)

type TaskStatus uint8

var ErrTaskIDEmpty = errors.New("InferenceTask.ID is 0")

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
	TaskFee          BigInt          `json:"task_fee"`
	TaskSize         uint64          `json:"task_size"`
	ModelIDs         StringArray     `json:"model_ids" gorm:"type:text"`
	AbortReason      TaskAbortReason `json:"abort_reason"`
	TaskError        TaskError       `json:"task_error"`
	Score            string          `json:"score" gorm:"type:text"`
	QOSScore         uint64          `json:"qos_score"`
	SelectedNode     string          `json:"selected_node"`
	TaskID           string          `json:"task_id"`
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

func (task *InferenceTask) VersionNumbers() [3]uint64 {
	taskVersions := strings.Split(task.TaskVersion, ".")
	if len(taskVersions) != 3 {
		log.Fatalf("Task version is invalid: %d", task.ID)
	}
	taskMajorVersion, err := strconv.ParseUint(taskVersions[0], 10, 64)
	if err != nil {
		log.Fatalf("Task version is invalid: %d", task.ID)
	}
	taskMinorVersion, err := strconv.ParseUint(taskVersions[1], 10, 64)
	if err != nil {
		log.Fatalf("Task version is invalid: %d", task.ID)
	}
	taskPatchVersion, err := strconv.ParseUint(taskVersions[2], 10, 64)
	if err != nil {
		log.Fatalf("Task version is invalid: %d", task.ID)
	}
	return [3]uint64{taskMajorVersion, taskMinorVersion, taskPatchVersion}
}

func (task *InferenceTask) SyncStatus(ctx context.Context, db *gorm.DB) error {
	if task.ID == 0 {
		return ErrTaskIDEmpty
	}
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	var res InferenceTask
	if err := db.WithContext(dbCtx).Model(task).Select("status").First(&res).Error; err != nil {
		return err
	}
	task.Status = res.Status
	log.Infof("task %d status %d", task.ID, task.Status)
	return nil
}

func (task *InferenceTask) Save(ctx context.Context, db *gorm.DB) error {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := db.WithContext(dbCtx).Save(task).Error; err != nil {
		return err
	}
	return nil
}

func (task *InferenceTask) Update(ctx context.Context, db *gorm.DB, values map[string]interface{}) error {
	if task.ID == 0 {
		return ErrTaskIDEmpty
	}
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := db.WithContext(dbCtx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(task).Updates(values).Error; err != nil {
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

func GetTaskByIDCommitment(ctx context.Context, db *gorm.DB, taskIDCommitment string) (*InferenceTask, error) {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	task := InferenceTask{TaskIDCommitment: taskIDCommitment}
	if err := db.WithContext(dbCtx).Model(&task).Where(&task).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func GetTaskGroupByTaskID(ctx context.Context, db *gorm.DB, taskID string) ([]InferenceTask, error) {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	var tasks []InferenceTask
	if err := db.WithContext(dbCtx).Model(&InferenceTask{}).Where("task_id = ?", taskID).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (task *InferenceTask) ExecutionTime() time.Duration {
	if task.StartTime.Valid && task.ScoreReadyTime.Valid {
		return task.ScoreReadyTime.Time.Sub(task.StartTime.Time)
	}
	return time.Duration(1<<63 - 1)
}
