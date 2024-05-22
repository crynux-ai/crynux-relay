package network

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"

	"github.com/gin-gonic/gin"
)

type AllTaskNumber struct {
	TotalTasks   uint64 `json:"total_tasks"`
	RunningTasks uint64 `json:"running_tasks"`
	QueuedTasks  uint64 `json:"queued_tasks"`
}

type GetAllTaskNumberResponse struct {
	response.Response
	Data *AllTaskNumber `json:"data"`
}

func GetAllTaskNumber(_ *gin.Context) (*GetAllTaskNumberResponse, error) {

	var taskNumber models.NetworkTaskNumber
	if err := config.GetDB().Model(&models.NetworkTaskNumber{}).First(&taskNumber).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetAllTaskNumberResponse{
		Data: &AllTaskNumber{
			TotalTasks:   taskNumber.TotalTasks,
			RunningTasks: taskNumber.RunningTasks,
			QueuedTasks:  taskNumber.QueuedTasks,
		},
	}, nil
}
