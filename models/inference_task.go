package models

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/gorm"
	"strconv"
)

type TaskStatus int

const (
	InferenceTaskCreatedOnChain TaskStatus = iota
	InferenceTaskParamsUploaded
	InferenceTaskAborted
	InferenceTaskPendingResults
	InferenceTaskResultsUploaded
)

type ChainTaskType int

const (
	TaskTypeSD ChainTaskType = iota
	TaskTypeLLM
)

type InferenceTask struct {
	gorm.Model
	TaskArgs      string        `json:"task_args"`
	TaskId        uint64        `json:"task_id" gorm:"index"`
	Creator       string        `json:"creator"`
	TaskHash      string        `json:"task_hash"`
	DataHash      string        `json:"data_hash"`
	Status        TaskStatus    `json:"status"`
	TaskType      ChainTaskType `json:"task_type"`
	VramLimit     uint64        `json:"vram_limit"`
	SelectedNodes []SelectedNode
}

func (t *InferenceTask) GetTaskIdAsString() string {
	return strconv.FormatUint(t.TaskId, 10)
}

func (t *InferenceTask) GetTaskHash() (*common.Hash, error) {
	hash := crypto.Keccak256Hash([]byte(t.TaskArgs))
	return &hash, nil
}

func (t *InferenceTask) GetDataHash() (*common.Hash, error) {
	return nil, nil
}
