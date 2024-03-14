package network

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/blockchain"
	"math/big"

	"github.com/gin-gonic/gin"
)

type AllTaskNumber struct {
	TotalTasks   *big.Int `json:"total_tasks"`
	RunningTasks *big.Int `json:"running_tasks"`
	QueuedTasks  *big.Int `json:"queued_tasks"`
}

type GetAllTaskNumberResponse struct {
	response.Response
	Data *AllTaskNumber `json:"data"`
}

func GetAllTaskNumber(_ *gin.Context) (*GetAllTaskNumberResponse, error) {

	totalTasks, runningTasks, queuedTasks, err := blockchain.GetAllTasksNumber()

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetAllTaskNumberResponse{
		Data: &AllTaskNumber{
			TotalTasks:   totalTasks,
			RunningTasks: runningTasks,
			QueuedTasks:  queuedTasks,
		},
	}, nil
}
