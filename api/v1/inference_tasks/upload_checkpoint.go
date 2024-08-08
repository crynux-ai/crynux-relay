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

type CheckpointInput struct {
	TaskId uint64 `path:"task_id" json:"task_id" description:"Task id" validate:"required"`
}

type CheckpointInputWithSignature struct {
	CheckpointInput
	Timestamp int64  `form:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `form:"signature" json:"signature" description:"Signature" validate:"required"`
}


func UploadCheckpoint(ctx *gin.Context, in *CheckpointInputWithSignature) (*response.Response, error) {
	match, address, err := ValidateSignature(in.CheckpointInput, in.Timestamp, in.Signature)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	if !match {
		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	var task models.InferenceTask

	if result := config.GetDB().Where(&models.InferenceTask{TaskId: in.TaskId}).First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("task_id", "Task not found on the Blockchain")
			return nil, validationErr
		} else {
			return nil, response.NewExceptionResponse(result.Error)
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
				"Task checkpoint already uploaded")
	}

	appConfig := config.GetConfig()
	taskWorkspace := appConfig.DataDir.InferenceTasks
	taskIdStr := task.GetTaskIdAsString()

	taskDir := filepath.Join(taskWorkspace, taskIdStr, "input")
	if err = os.MkdirAll(taskDir, os.ModePerm); err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	form, _ := ctx.MultipartForm()
	checkpointFiles, ok := form.File["checkpoint"]
	if !ok {
		return nil, response.NewValidationErrorResponse("checkpoint", "Checkpoint not uploaded")
	}
	if len(checkpointFiles) != 1 {
		return nil, response.NewValidationErrorResponse("checkpoint", "Too many checkpoint files")
	}
	checkpointFile := checkpointFiles[0]
	checkpointFilename := filepath.Join(taskDir, "checkpoint.zip")
	if err := ctx.SaveUploadedFile(checkpointFile, checkpointFilename); err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	return &response.Response{}, nil
}
