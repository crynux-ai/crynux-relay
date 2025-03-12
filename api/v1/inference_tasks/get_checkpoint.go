package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/validate"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetCheckpointInput struct {
	TaskIDCommitment   string `path:"task_id_commitment" json:"task_id_commitment" description:"Task id commitment" validate:"required"`
}

type GetCheckpointInputWithSignature struct {
	GetCheckpointInput
	Timestamp int64  `query:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" description:"Signature" validate:"required"`
}

func GetCheckpoint(c *gin.Context, in *GetCheckpointInputWithSignature) error {
	match, address, err := validate.ValidateSignature(in.GetCheckpointInput, in.Timestamp, in.Signature)

	if err != nil || !match {
		return response.NewValidationErrorResponse("signature", "Invalid signature")
	}

	task, err := models.GetTaskByIDCommitment(c.Request.Context(), config.GetDB(), in.TaskIDCommitment)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("task_id_commitment", "Task not found")
			return validationErr
		} else {
			return response.NewExceptionResponse(err)
		}
	}

	if task.Status == models.TaskQueued {
		return response.NewValidationErrorResponse("task_id", "Task not ready")
	}

	if task.Creator != address && task.SelectedNode != address {
		return response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	appConfig := config.GetConfig()
	resultFile := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.TaskIDCommitment,
		"input",
		"checkpoint.zip",
	)
	if _, err := os.Stat(resultFile); err != nil {
		return response.NewValidationErrorResponse("task_id", "Checkpoint file not found")
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename=checkpoint.zip")
	c.Header("Content-Type", "application/octet-stream")
	c.File(resultFile)
	return nil
}