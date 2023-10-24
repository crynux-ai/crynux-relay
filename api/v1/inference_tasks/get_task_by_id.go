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

type GetTaskInput struct {
	TaskId uint64 `path:"task_id" json:"task_id" validate:"required" description:"The task id"`
}

type GetTaskInputWithSignature struct {
	GetTaskInput
	Timestamp int64  `query:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" json:"signature" description:"Signature" validate:"required"`
}

func GetTaskById(_ *gin.Context, in *GetTaskInputWithSignature) (*TaskResponse, error) {

	match, address, err := ValidateSignature(in.GetTaskInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	var task models.InferenceTask

	if result := config.GetDB().Where(&models.InferenceTask{TaskId: in.TaskId}).Preload("SelectedNodes").First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("task_id", "Task not found")
			return nil, validationErr
		} else {
			return nil, response.NewExceptionResponse(result.Error)
		}
	}

	if task.Status >= models.InferenceTaskParamsUploaded {
		return nil, response.NewValidationErrorResponse("task_id", "Task not ready")
	}

	if task.Creator == address {
		return &TaskResponse{Data: task}, nil
	}

	log.Debugln("signer address: " + address)
	log.Debugln("selected nodes")
	log.Debugln(task.SelectedNodes)

	for _, selectedNode := range task.SelectedNodes {
		if selectedNode.NodeAddress == address {
			return &TaskResponse{Data: task}, nil
		}
	}

	validationErr := response.NewValidationErrorResponse("signature", "Signer not allowed")
	return nil, validationErr
}
