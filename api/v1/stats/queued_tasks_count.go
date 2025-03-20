package stats

import (
	"context"
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	"github.com/gin-gonic/gin"
)

type QueuedTasksCountResponse struct {
	response.Response
	Data int64 `json:"data"`
}

func GetQueuedTasksCount(c *gin.Context) (*QueuedTasksCountResponse, error) {
	var cnt int64
	dbCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	if err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).Where("status = ?", models.TaskQueued).Count(&cnt).Error; err != nil {
		return nil, err
	}
	return &QueuedTasksCountResponse{
		Data: cnt,
	}, nil
}
