package inference_tasks

import (
	"context"
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/validate"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type GetTaskInput struct {
	TaskIDCommitment string `path:"task_id_commitment" json:"task_id_commitment" validate:"required" description:"The task id commitment"`
}

type GetTaskInputWithSignature struct {
	GetTaskInput
	Timestamp int64  `query:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" json:"signature" description:"Signature" validate:"required"`
}

func GetTaskById(c *gin.Context, in *GetTaskInputWithSignature) (*TaskResponse, error) {

	match, address, err := validate.ValidateSignature(in.GetTaskInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	var task models.InferenceTask

	dbCtx, dbCancel := context.WithTimeout(c.Request.Context(), time.Second)
	defer dbCancel()

	if result := config.GetDB().WithContext(dbCtx).Where(&models.InferenceTask{TaskIDCommitment: in.TaskIDCommitment}).First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("task_id", "Task not found")
			return nil, validationErr
		} else {
			return nil, response.NewExceptionResponse(result.Error)
		}
	}

	if len(task.TaskArgs) == 0 {
		return nil, response.NewValidationErrorResponse("task_id", "Task not ready")
	}

	if task.Status != models.InferenceTaskCreated && task.Status != models.InferenceTaskParamsUploaded {
		return nil, response.NewValidationErrorResponse("task_id", "Task not ready")
	}

	if task.SelectedNode != address && task.Creator != address {
		return nil, response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	return &TaskResponse{Data: task}, nil
}
