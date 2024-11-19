package models

import (
	"time"

	"gorm.io/gorm"
)

type TaskStatus uint8

const (
	InferenceTaskCreated TaskStatus = iota
	InferenceTaskParamsUploaded 
	InferenceTaskResultsUploaded
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
	StartTime          time.Time       `json:"start_time"`
	FinishTime         time.Time       `json:"finish_time"`
	ResultUploadedTime time.Time       `json:"result_uploaded_time"`
}
