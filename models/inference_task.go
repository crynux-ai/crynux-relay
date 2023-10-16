package models

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strconv"
)

type TaskStatus int

const (
	InferenceTaskCreatedOnChain TaskStatus = iota
	InferenceTaskUploaded
)

type InferenceTask struct {
	gorm.Model
	TaskArgs      string     `json:"task_args"`
	TaskId        uint64     `json:"task_id"`
	Creator       string     `json:"creator"`
	TaskHash      string     `json:"task_hash"`
	DataHash      string     `json:"data_hash"`
	Status        TaskStatus `json:"status"`
	SelectedNodes []SelectedNode
}

func (t *InferenceTask) GetTaskIdAsString() string {
	return strconv.FormatUint(t.TaskId, 10)
}

func (t *InferenceTask) GetTaskHash() (*common.Hash, error) {

	taskHashBytes, err := json.Marshal(t.TaskArgs)
	if err != nil {
		return nil, err
	}

	log.Debugln("task hash string: " + string(taskHashBytes))

	hash := crypto.Keccak256Hash(taskHashBytes)
	return &hash, nil
}

func (t *InferenceTask) GetDataHash() (*common.Hash, error) {
	return nil, nil
}
