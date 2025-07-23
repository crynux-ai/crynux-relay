package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/validate"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"
	"errors"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ReportTaskErrorInput struct {
	TaskIDCommitment string           `path:"task_id_commitment" json:"task_id_commitment" description:"Task id commitment" validate:"required"`
	TaskError        models.TaskError `json:"task_error" description:"Task error" validate:"required"`
}

type ReportTaskErrorInputWithSignature struct {
	ReportTaskErrorInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func ReportTaskError(c *gin.Context, in *ReportTaskErrorInputWithSignature) (*response.Response, error) {
	match, address, err := validate.ValidateSignature(in.ReportTaskErrorInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	task, err := models.GetTaskByIDCommitment(c.Request.Context(), config.GetDB(), in.TaskIDCommitment)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("task_id_commitment", "Task not found")
			return nil, validationErr
		} else {
			return nil, response.NewExceptionResponse(err)
		}
	}

	if task.Status != models.TaskStarted {
		return nil, response.NewValidationErrorResponse("task_id_commitment", "Illegal task state")
	}

	if task.SelectedNode != address {
		return nil, response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	task.TaskError = in.TaskError
	for range 3 {
		err = service.SetTaskStatusErrorReported(c.Request.Context(), config.GetDB(), task)
		if err == nil {
			break
		} else if errors.Is(err, models.ErrTaskStatusChanged) || errors.Is(err, models.ErrNodeStatusChanged) {
			if err := task.SyncStatus(c.Request.Context(), config.GetDB()); err != nil {
				return nil, response.NewExceptionResponse(err)
			}
		} else {
			return nil, response.NewExceptionResponse(err)
		}
	}
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	return &response.Response{}, nil
}
