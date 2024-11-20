package inference_tasks

import (
	"crynux_relay/api/v1/response"
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

func GetCheckpoint(ctx *gin.Context, in *GetCheckpointInputWithSignature) error {
	match, address, err := ValidateSignature(in.GetCheckpointInput, in.Timestamp, in.Signature)

	if err != nil || !match {
		return response.NewValidationErrorResponse("signature", "Invalid signature")
	}

	var task models.InferenceTask

	if result := config.GetDB().Where(&models.InferenceTask{TaskIDCommitment: in.TaskIDCommitment}).First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("task_id", "Task not found")
			return validationErr
		} else {
			return response.NewExceptionResponse(result.Error)
		}
	}

	if task.Status != models.InferenceTaskParamsUploaded {
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

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Content-Disposition", "attachment; filename=checkpoint.zip")
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.File(resultFile)
	return nil
}