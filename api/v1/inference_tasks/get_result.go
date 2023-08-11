package inference_tasks

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/config"
	"h_relay/models"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type GetResultInput struct {
	TaskId       int64  `path:"task_id" json:"task_id" description:"Task id" validate:"required"`
	SelectedNode string `path:"node" json:"node" description:"Selected node" validate:"required"`
	ImageNum     int    `path:"image_num" json:"image_num" description:"The image number" validate:"required"`
}

type GetResultInputWithSignature struct {
	GetResultInput
	Timestamp int64  `query:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" json:"signature" description:"Signature" validate:"required"`
}

func GetResult(ctx *gin.Context, in *GetResultInputWithSignature) {

	sigStr, err := json.Marshal(&GetResultInput{
		TaskId:       in.TaskId,
		SelectedNode: in.SelectedNode,
		ImageNum:     in.ImageNum,
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Internal server error",
		})
		return
	}

	match, address, err := ValidateSignature(sigStr, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln(err)
		}

		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid signature",
		})

		return
	}

	var task models.InferenceTask

	if result := config.GetDB().Where(&models.InferenceTask{TaskId: in.TaskId}).First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "Task not found",
			})
			return
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": "Internal server error",
			})
			return
		}
	}

	if task.Creator != address {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Signer not allowed",
		})
		return
	}

	appConfig := config.GetConfig()
	imageFile := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.GetTaskIdAsString(),
		in.SelectedNode,
		strconv.Itoa(in.ImageNum)+".png",
	)

	if _, err := os.Stat(imageFile); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "File not found",
		})
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Content-Disposition", "attachment; filename="+strconv.Itoa(in.ImageNum)+".png")
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.File(imageFile)
}
