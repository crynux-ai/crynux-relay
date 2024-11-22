package models

import (
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
	InferenceTaskEndSuccess
	InferenceTaskEndInvalidated
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
	TaskArgs           string          `json:"task_args"`
	TaskIDCommitment   string          `json:"task_id_commitment" gorm:"index"`
	Creator            string          `json:"creator"`
	Status             TaskStatus      `json:"status"`
	TaskType           ChainTaskType   `json:"task_type"`
	MinVRAM            uint64          `json:"min_vram"`
	RequiredGPU        string          `json:"required_gpu"`
	RequiredGPUVRAM    uint64          `json:"required_gpu_vram"`
	TaskFee            float64         `json:"task_fee"`
	TaskSize           uint64          `json:"task_size"`
	AbortReason        TaskAbortReason `json:"abort_reason"`
	TaskError          TaskError       `json:"task_error"`
	SelectedNode       string          `json:"selected_node"`
	// time when task is created (get from blockchain)
	CreateTime         time.Time       `json:"create_time"`
	// time when relay report task params are uploaded
	StartTime          time.Time       `json:"start_time"`
	// time when task score is ready (get from blockchain)
	ScoreReadyTime     time.Time       `json:"score_ready_time"`
	// time when relay find that task score is validated
	ValidatedTime      time.Time       `json:"validated_time"`
	// time when relay report task results are uploaded
	ResultUploadedTime time.Time       `json:"result_uploaded_time"`
}
