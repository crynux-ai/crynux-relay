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
	TaskId int64 `query:"task_id" json:"task_id" validate:"required" description:"The task id"`
}

type GetTaskInputWithSignature struct {
	GetTaskInput
	Signer    string `query:"signer" json:"signer" description:"Signer address" validate:"required"`
	Timestamp int64  `query:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" json:"signature" description:"Signature" validate:"required"`
}

func GetTaskById(ctx *gin.Context, in *GetTaskInputWithSignature) (*TaskResponse, error) {
	sigStr, err := json.Marshal(in.GetTaskInput)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	match, err := ValidateSignature(in.Signer, sigStr, in.Timestamp, in.Signature)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	if !match {
		validationErr := response.NewValidationErrorResponse()
		validationErr.SetFieldName("signature")
		validationErr.SetFieldMessage("Invalid signature")
		return nil, validationErr
	}

	var task models.InferenceTask

	if result := config.GetDB().Where(&models.InferenceTask{TaskId: in.TaskId}).First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse()
			validationErr.SetFieldName("task_id")
			validationErr.SetFieldMessage("Task not found")

			return nil, validationErr
		} else {
			return nil, response.NewExceptionResponse(result.Error)
		}
	}

	if task.Creator == in.Signer {
		return &TaskResponse{Data: task}, nil
	}

	var selectedNodes []string

	if err = json.Unmarshal([]byte(task.SelectedNodes), &selectedNodes); err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	for _, nodeAddr := range selectedNodes {
		if nodeAddr == in.Signer {
			return &TaskResponse{Data: task}, nil
		}
	}

	validationErr := response.NewValidationErrorResponse()
	validationErr.SetFieldName("signer")
	validationErr.SetFieldMessage("Signer not allowed")

	return nil, validationErr
}
