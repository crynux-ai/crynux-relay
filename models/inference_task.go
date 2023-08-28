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

type TaskConfig struct {
	ImageHeight int `json:"image_height" description:"Image height" validate:"required,lte=1024"`
	ImageWidth  int `json:"image_width" description:"Image width" validate:"required,lte=1024"`
	LoraWeight  int `json:"lora_weight" description:"Weight of the LoRA model" validate:"required,max=100,min=1"`
	NumImages   int `json:"num_images" description:"Number of images to generate" validate:"required,min=1,max=9"`
	Seed        int `json:"seed" description:"The random seed used to generate images" validate:"required"`
	Steps       int `json:"steps" description:"Steps" validate:"required,max=100,min=10"`
}

type PoseConfig struct {
	DataURL    string `json:"data_url" description:"The pose image DataURL" default:""`
	PoseWeight int    `json:"pose_weight" description:"Weight of the pose model" validate:"required,max=100,min=1"`
	Preprocess bool   `json:"preprocess" description:"Preprocess the image"`
}

type InferenceTask struct {
	gorm.Model
	TaskId        uint64      `json:"task_id"`
	Creator       string      `json:"creator"`
	TaskHash      string      `json:"task_hash"`
	DataHash      string      `json:"data_hash"`
	Prompt        string      `json:"prompt"`
	BaseModel     string      `json:"base_model"`
	LoraModel     string      `json:"lora_model"`
	TaskConfig    *TaskConfig `json:"task_config" gorm:"embedded"` // Before params uploaded, the field will be empty
	Pose          *PoseConfig `json:"pose" gorm:"embedded"`        // Before params uploaded, the field will be empty
	Status        TaskStatus  `json:"status"`
	SelectedNodes []SelectedNode
}

func (t *InferenceTask) GetTaskIdAsString() string {
	return strconv.FormatUint(t.TaskId, 10)
}

type DataHashInput struct {
	BaseModel string     `json:"base_model"`
	LoraModel string     `json:"lora_model"`
	Pose      PoseConfig `json:"pose"`
	Prompt    string     `json:"prompt"`
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
		BaseModel: t.BaseModel,
		LoraModel: t.LoraModel,
		Prompt:    t.Prompt,
		Pose:      *t.Pose,
	}

	dataHashBytes, err := json.Marshal(dataHash)
	if err != nil {
		return nil, err
	}

	log.Debugln("data hash string: " + string(dataHashBytes))

	hash := crypto.Keccak256Hash(dataHashBytes)
	return &hash, nil
}
