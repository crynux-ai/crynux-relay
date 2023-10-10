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
	TaskArgs      `gorm:"embedded;embeddedPrefix:task_args_"`
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

type DataHashInput struct {
	BaseModel      string          `json:"base_model"`
	Controlnet     *ControlnetArgs `json:"controlnet"`
	Lora           *LoraArgs       `json:"lora"`
	NegativePrompt string          `json:"negative_prompt"`
	Prompt         string          `json:"prompt"`
	Refiner        *RefinerArgs    `json:"refiner"`
	VAE            string          `json:"vae"`
}

func (t *InferenceTask) GetTaskHash() (*common.Hash, error) {

	taskHashBytes, err := json.Marshal(t.TaskConfig)
	if err != nil {
		return nil, err
	}

	log.Debugln("task hash string: " + string(taskHashBytes))

	hash := crypto.Keccak256Hash(taskHashBytes)
	return &hash, nil
}

func (t *InferenceTask) GetDataHash() (*common.Hash, error) {

	dataHash := &DataHashInput{
		BaseModel:      t.BaseModel,
		Controlnet:     t.Controlnet,
		Lora:           t.Lora,
		NegativePrompt: t.NegativePrompt,
		Prompt:         t.Prompt,
		Refiner:        t.Refiner,
		VAE:            t.VAE,
	}

	dataHashBytes, err := json.Marshal(dataHash)
	if err != nil {
		return nil, err
	}

	log.Debugln("data hash string: " + string(dataHashBytes))

	hash := crypto.Keccak256Hash(dataHashBytes)
	return &hash, nil
}
