package inference_tasks

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"h_relay/api/v1/response"
	"h_relay/config"
	"h_relay/models"
)

type GetTaskInput struct {
	TaskId int64 `path:"task_id" json:"task_id" validate:"required" description:"The task id"`
}

type GetTaskInputWithSignature struct {
	GetTaskInput
	Timestamp int64  `query:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" json:"signature" description:"Signature" validate:"required"`
}

func GetTaskById(ctx *gin.Context, in *GetTaskInputWithSignature) (*TaskResponse, error) {

	match, address, err := ValidateSignature(in.GetTaskInput, in.Timestamp, in.Signature)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	if !match {
		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	var task models.InferenceTask

	if result := config.GetDB().Where(&models.InferenceTask{TaskId: uint64(in.TaskId)}).First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("task_id", "Task not found")
			return nil, validationErr
		} else {
			return nil, response.NewExceptionResponse(result.Error)
		}
	}

	if task.Creator == address {
		return &TaskResponse{Data: task}, nil
	}

	var selectedNodes []string

	if err = json.Unmarshal([]byte(task.SelectedNodes), &selectedNodes); err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	for _, nodeAddr := range selectedNodes {
		if nodeAddr == address {
			return &TaskResponse{Data: task}, nil
		}
	}

	validationErr := response.NewValidationErrorResponse("signature", "Signer not allowed")
	return nil, validationErr
}
