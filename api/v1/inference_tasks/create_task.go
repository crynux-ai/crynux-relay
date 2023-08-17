package inference_tasks

import (
	"errors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/api/v1/response"
	"h_relay/config"
	"h_relay/models"
)

type TaskInput struct {
	BaseModel  string            `json:"base_model" validate:"required"`
	LoraModel  string            `json:"lora_model" default:""`
	Pose       models.PoseConfig `json:"pose" validate:"required"`
	Prompt     string            `json:"prompt" validate:"required"`
	TaskConfig models.TaskConfig `json:"task_config"`
	TaskId     uint64            `json:"task_id" description:"Task id" validate:"required"`
}

type TaskInputWithSignature struct {
	TaskInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func CreateTask(_ *gin.Context, in *TaskInputWithSignature) (*TaskResponse, error) {

	match, address, err := ValidateSignature(in.TaskInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	task := models.InferenceTask{
		TaskId: in.TaskId,
	}

	if err := config.GetDB().Where(task).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil,
				response.NewValidationErrorResponse(
					"task_id",
					"Task not found on the Blockchain")
		} else {
			return nil, response.NewExceptionResponse(err)
		}
	}

	if task.Creator != address {
		return nil,
			response.NewValidationErrorResponse(
				"signature",
				"Signer not allowed")
	}

	if task.Status != models.InferenceTaskCreatedOnChain {
		return nil,
			response.NewValidationErrorResponse(
				"task_id",
				"Task already uploaded")
	}

	task.BaseModel = in.BaseModel
	task.LoraModel = in.LoraModel
	task.Prompt = in.Prompt
	task.TaskConfig = &in.TaskConfig
	task.Pose = &in.Pose
	task.Status = models.InferenceTaskUploaded

	taskHash, err := task.GetTaskHash()
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if taskHash != task.TaskHash {
		return nil,
			response.NewValidationErrorResponse(
				"task_hash",
				"Task hash mismatch")
	}

	dataHash, err := task.GetDataHash()
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if dataHash != task.DataHash {
		return nil,
			response.NewValidationErrorResponse(
				"data_hash",
				"Data hash mismatch")
	}

	if err := config.GetDB().Save(&task).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &TaskResponse{Data: task}, nil
}
