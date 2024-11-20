package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type GetResultInput struct {
	Index            string `path:"index" json:"index" description:"Result file index" validate:"required"`
	TaskIDCommitment string `path:"task_id_commitment" json:"task_id_commitment" description:"Task id commitment" validate:"required"`
}

type GetResultInputWithSignature struct {
	GetResultInput
	Timestamp int64  `query:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" description:"Signature" validate:"required"`
}

func GetResult(ctx *gin.Context, in *GetResultInputWithSignature) error {

	match, address, err := ValidateSignature(in.GetResultInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln(err)
		}

		return response.NewValidationErrorResponse("signature", "Invalid signature")
	}

	var task models.InferenceTask

	if err := config.GetDB().Where(&models.InferenceTask{TaskIDCommitment: in.TaskIDCommitment}).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewValidationErrorResponse("task_id", "Task not found")
		} else {
			return response.NewExceptionResponse(err)
		}
	}

	if task.Creator != address {
		return response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	if task.Status != models.InferenceTaskEndSuccess {
		return response.NewValidationErrorResponse("task_id", "Task results not uploaded")
	}

	appConfig := config.GetConfig()

	var fileExt string
	if task.TaskType == models.TaskTypeSD || task.TaskType == models.TaskTypeSDFTLora {
		fileExt = ".png"
	} else {
		fileExt = ".json"
	}

	resultFile := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.TaskIDCommitment,
		"results",
		in.Index+fileExt,
	)

	if _, err := os.Stat(resultFile); err != nil {
		return response.NewValidationErrorResponse("index", "File not found")
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Content-Disposition", "attachment; filename="+in.Index+fileExt)
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.File(resultFile)

	return nil
}

type GetResultCheckpointInput struct {
	TaskIDCommitment string `path:"task_id_commitment" json:"task_id_commitment" description:"Task id commitment" validate:"required"`
}

type GetResultCheckpointInputWithSignature struct {
	GetResultCheckpointInput
	Timestamp int64  `query:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" description:"Signature" validate:"required"`
}

func GetResultCheckpoint(ctx *gin.Context, in *GetResultCheckpointInputWithSignature) error {
	match, address, err := ValidateSignature(in.GetResultCheckpointInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln(err)
		}

		return response.NewValidationErrorResponse("signature", "Invalid signature")
	}

	var task models.InferenceTask

	if err := config.GetDB().Where(&models.InferenceTask{TaskIDCommitment: in.TaskIDCommitment}).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewValidationErrorResponse("task_id", "Task not found")
		} else {
			return response.NewExceptionResponse(err)
		}
	}

	if task.Creator != address {
		return response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	if task.Status != models.InferenceTaskEndSuccess {
		return response.NewValidationErrorResponse("task_id", "Task checkpoint not uploaded")
	}

	appConfig := config.GetConfig()
	resultFile := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.TaskIDCommitment,
		"results",
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
