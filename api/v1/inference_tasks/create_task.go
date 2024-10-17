package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type TaskInput struct {
	TaskArgs string `json:"task_args" description:"Task arguments" validate:"required"`
	TaskId   uint64 `json:"task_id" description:"Task id" validate:"required"`
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

	validationErr, err := models.ValidateTaskArgsJsonStr(in.TaskArgs, task.TaskType)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	if validationErr != nil {
		return nil, response.NewValidationErrorResponse("task_args", validationErr.Error())
	}

	if task.Creator != address {
		return nil,
			response.NewValidationErrorResponse(
				"signature",
				"Signer not allowed")
	}

	if len(task.TaskArgs) > 0 {
		return nil,
			response.NewValidationErrorResponse(
				"task_id",
				"Task already uploaded")
	}

	task.TaskArgs = in.TaskArgs
	task.Status = models.InferenceTaskParamsUploaded

	taskStatusLog := models.InferenceTaskStatusLog{
		InferenceTask: task,
		Status: models.InferenceTaskParamsUploaded,
	}

	taskHash, err := task.GetTaskHash()
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if taskHash.Hex() != task.TaskHash {
		return nil,
			response.NewValidationErrorResponse(
				"task_hash",
				"Task hash mismatch")
	}

	err = config.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&task).Error; err != nil {
			return err
		}
		if err := tx.Create(&taskStatusLog).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &TaskResponse{Data: task}, nil
}
