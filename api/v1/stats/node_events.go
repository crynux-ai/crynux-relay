package stats

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetNodeEventLogs(ctx *gin.Context) error {
	start := time.Now().Add(-time.Hour)

	subquery := config.GetDB().Table("selected_nodes").Select("selected_nodes.id AS id, selected_nodes.node_address AS node_address, inference_tasks.task_id AS task_id, inference_tasks.task_args AS task_args").Joins("INNER JOIN inference_tasks ON selected_nodes.inference_task_id = inference_tasks.id AND inference_tasks.task_type = 0")
	rows, err := config.GetDB().Table("selected_node_status_logs").Select("selected_node_status_logs.created_at, selected_node_status_logs.status, s.node_address, s.task_id, s.task_args").Joins("(?) s ON selected_node_status_logs.selected_node_id = s.id", subquery).Where("selected_node_status_logs.created_at >= ? AND selected_node_status_logs.status != ?", start, models.NodeStatusRunning).Rows()
	if err != nil {
		return response.NewExceptionResponse(err)
	}
	defer rows.Close()

	var t time.Time
	var status models.NodeStatus
	var nodeAddress string
	var taskID uint64
	var taskArgs string

	var times []time.Time
	var allStatus []models.NodeStatus
	var nodeAddresses []string
	var taskIDs []uint64
	var taskModels []string
	for rows.Next() {
		if err := rows.Scan(&t, &status, &nodeAddress, &taskID, &taskArgs); err != nil {
			return response.NewExceptionResponse(err)
		}

		times = append(times, t)
		allStatus = append(allStatus, status)
		nodeAddresses = append(nodeAddresses, nodeAddress)
		taskIDs = append(taskIDs, taskID)
		if strings.Contains(taskArgs, "crynux-ai/stable-diffusion-xl-base-1.0") {
			taskModels = append(taskModels, "SDXL")
		} else {
			taskModels = append(taskModels, "SD1.5")
		}
	}

	var nodeDatas []models.NetworkNodeData
	if err := config.GetDB().Model(&models.NetworkNodeData{}).Where("address IN (?)", nodeAddresses).Find(&nodeDatas).Error; err != nil {
		return response.NewExceptionResponse(err)
	}

	nodeDataMap := make(map[string]models.NetworkNodeData)
	for _, nodeData := range nodeDatas {
		nodeDataMap[nodeData.Address] = nodeData
	}

	var builder strings.Builder
	length := len(times)
	for i := 0; i < length; i++ {
		timeString := times[i].UTC().Format(time.RFC3339)
		var statusString string
		if allStatus[i] == models.NodeStatusPending {
			statusString = "Node Selected"
		} else {
			statusString = "Node Released"
		}
		nodeAddress := nodeAddresses[i]
		nodeData := nodeDataMap[nodeAddress]
		taskModel := taskModels[i]
		taskID := taskIDs[i]
		fmt.Fprintf(&builder, "[%s] [%s] [%s.%d] [%s] [%s] [%d]\n", timeString, statusString, nodeData.CardModel, nodeData.VRam, nodeAddress, taskModel, taskID)
	}

	ctx.String(http.StatusOK, builder.String())
	return nil
}
