package models

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/gorm"
	"strconv"
)

type TaskStatus int

const (
	InferenceTaskCreatedOnChain TaskStatus = iota
	InferenceTaskUploaded
)

type TaskConfig struct {
	ImageWidth  int     `json:"image_width" description:"Image width" validate:"required,lte=1024"`
	ImageHeight int     `json:"image_height" description:"Image height" validate:"required,lte=1024"`
	Steps       int     `json:"steps" description:"Steps" validate:"required,max=100,min=10"`
	LoraWeight  float32 `json:"lora_weight" description:"Weight of the LoRA model" validate:"required,max=1,min=0.1"`
	PoseWeight  float32 `json:"pose_weight" description:"Weight of the pose model" validate:"required,max=1,min=0.1"`
	NumImages   int     `json:"num_images" description:"Number of images to generate" validate:"required,min=1,max=9"`
	Seed        int     `json:"seed" description:"The random seed used to generate images" validate:"required,min=1,max=9"`
}

type PoseConfig struct {
	DataURL    string `json:"data_url" description:"The pose image DataURL" default:""`
	Preprocess bool   `json:"preprocess" description:"Preprocess the image" validate:"required"`
}

type InferenceTask struct {
	gorm.Model
	TaskId         uint64     `json:"task_id"`
	Creator        string     `json:"creator"`
	TaskHash       string     `json:"task_hash"`
	DataHash       string     `json:"data_hash"`
	Prompt         string     `json:"prompt"`
	BaseModel      string     `json:"base_model"`
	LoraModel      string     `json:"lora_model"`
	TaskConfig     TaskConfig `json:"task_config" gorm:"embedded"`
	PosePreprocess bool       `json:"pose_preprocess"`

	Status TaskStatus `json:"status"`

	SelectedNodes []SelectedNode
}

func (t *InferenceTask) GetTaskIdAsString() string {
	return strconv.FormatInt(int64(t.TaskId), 10)
}

type DataHashInput struct {
	Prompt         string `json:"prompt"`
	BaseModel      string `json:"base_model"`
	LoraModel      string `json:"lora_model"`
	PosePreprocess bool   `json:"pose_preprocess"`
}

func (t *InferenceTask) GetTaskHash() (string, error) {

	taskHashBytes, err := json.Marshal(t.TaskConfig)
	if err != nil {
		return "", err
	}

	hash := crypto.Keccak256Hash(taskHashBytes)
	return hash.Hex(), nil
}

func (t *InferenceTask) GetDataHash() (string, error) {

	dataHash := &DataHashInput{
		Prompt:         t.Prompt,
		BaseModel:      t.BaseModel,
		LoraModel:      t.LoraModel,
		PosePreprocess: t.PosePreprocess,
	}

	dataHashBytes, err := json.Marshal(dataHash)
	if err != nil {
		return "", err
	}

	hash := crypto.Keccak256Hash(dataHashBytes)
	return hash.Hex(), nil
}
