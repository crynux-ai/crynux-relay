package inference_tasks

import (
	"errors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/api/v1/response"
	"h_relay/config"
	"h_relay/models"
	"os"
	"path/filepath"
)

type GetResultInput struct {
	ImageNum string `path:"image_num" json:"image_num" description:"Image number" validate:"required"`
	TaskId   uint64 `path:"task_id" json:"task_id" description:"Task id" validate:"required"`
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

	appConfig := config.GetConfig()
	imageFile := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.GetTaskIdAsString(),
		"results",
		in.ImageNum+".png",
	)

	if _, err := os.Stat(imageFile); err != nil {
		return response.NewValidationErrorResponse("image_num", "File not found")
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Content-Disposition", "attachment; filename="+in.ImageNum+".png")
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.File(imageFile)

	return nil
}
