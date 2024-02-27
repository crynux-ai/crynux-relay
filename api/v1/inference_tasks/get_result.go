package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type GetSDResultInput struct {
	ImageNum string `path:"image_num" json:"image_num" description:"Image number" validate:"required"`
	TaskId   uint64 `path:"task_id" json:"task_id" description:"Task id" validate:"required"`
}

type GetSDResultInputWithSignature struct {
	GetSDResultInput
	Timestamp int64  `query:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" description:"Signature" validate:"required"`
}

func GetSDResult(ctx *gin.Context, in *GetSDResultInputWithSignature) error {

	match, address, err := ValidateSignature(in.GetSDResultInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln(err)
		}

		return response.NewValidationErrorResponse("signature", "Invalid signature")
	}

	var task models.InferenceTask

	if err := config.GetDB().Where(&models.InferenceTask{TaskId: in.TaskId}).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewValidationErrorResponse("task_id", "Task not found")
		} else {
			return response.NewExceptionResponse(err)
		}
	}

	if task.Creator != address {
		return response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	if task.Status != models.InferenceTaskResultsUploaded {
		return response.NewValidationErrorResponse("task_id", "Task results not uploaded")
	}

	appConfig := config.GetConfig()

	resultFile := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.GetTaskIdAsString(),
		"results",
		in.ImageNum+".png",
	)

	if _, err := os.Stat(resultFile); err != nil {
		return response.NewValidationErrorResponse("image_num", "File not found")
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Content-Disposition", "attachment; filename="+in.ImageNum+".png")
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.File(resultFile)

	return nil
}

type GetGPTResultInput struct {
	TaskId uint64 `path:"task_id" json:"task_id" description:"Task id" validate:"required"`
}

type GetGPTResultInputWithSignature struct {
	GetGPTResultInput
	Timestamp int64  `query:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" description:"Signature" validate:"required"`
}

func GetGPTResult(ctx *gin.Context, in *GetGPTResultInputWithSignature) (*GPTResultResponse, error) {
	match, address, err := ValidateSignature(in.GetGPTResultInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln(err)
		}

		return nil, response.NewValidationErrorResponse("signature", "Invalid signature")
	}

	var task models.InferenceTask

	if err := config.GetDB().Where(&models.InferenceTask{TaskId: in.TaskId}).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewValidationErrorResponse("task_id", "Task not found")
		} else {
			return nil, response.NewExceptionResponse(err)
		}
	}

	if task.Creator != address {
		return nil, response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	if task.Status != models.InferenceTaskResultsUploaded {
		return nil, response.NewValidationErrorResponse("task_id", "Task results not uploaded")
	}

	appConfig := config.GetConfig()

	resultFile := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.GetTaskIdAsString(),
		"results",
		"0.json",
	)

	if _, err := os.Stat(resultFile); err != nil {
		return nil, response.NewValidationErrorResponse("image_num", "File not found")
	}

	resultContent, err := os.ReadFile(resultFile)
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	data := &models.GPTTaskResponse{}
	if err := json.Unmarshal(resultContent, data); err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GPTResultResponse{Data: *data}, nil
}
