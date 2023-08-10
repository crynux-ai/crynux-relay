package inference_tasks

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
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
	SelectedNode string `query:"selected_node" json:"selected_node" description:"Selected node" validate:"required"`
	ImageNum     int    `path:"image_num" json:"image_num" description:"The image number" validate:"required"`
}

func GetResult(ctx *gin.Context) {

	taskIdStr := ctx.Param("task_id")
	imageNumStr := ctx.Param("image_num")
	selectedNode := ctx.Query("selected_node")

	timestampStr := ctx.Query("timestamp")
	signature := ctx.Query("signature")

	if taskIdStr == "" || imageNumStr == "" || selectedNode == "" || timestampStr == "" || signature == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Missing arguments",
		})
		return
	}

	taskId, err := strconv.ParseInt(taskIdStr, 10, 64)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid arguments",
		})
		return
	}

	imageNum, err := strconv.Atoi(imageNumStr)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid arguments",
		})
		return
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid arguments",
		})
		return
	}

	sigStr, err := json.Marshal(&GetResultInput{
		TaskId:       taskId,
		SelectedNode: selectedNode,
		ImageNum:     imageNum,
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Internal server error",
		})
		return
	}

	match, address, err := ValidateSignature(sigStr, timestamp, signature)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Internal server error",
		})
		return
	}

	if !match {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid signature",
		})
		return
	}

	var task models.InferenceTask

	if result := config.GetDB().Where(&models.InferenceTask{TaskId: taskId}).First(&task); result.Error != nil {
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
		selectedNode,
		imageNumStr+".jpg",
	)

	if _, err := os.Stat(imageFile); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "File not found",
		})
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Content-Disposition", "attachment; filename="+imageNumStr+".jpg")
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.File(imageFile)
}
