package incentive

import (
	"crynux_relay/api/v2/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"
	"time"

	"github.com/gin-gonic/gin"
)

type GetNodeIncentiveParams struct {
	Period TimeUnit `query:"period" validate:"required" enum:"Day,Week,Month"`
	Size   int      `query:"size" default:"30"`
}

type NodeIncentive struct {
	NodeAddress string  `json:"node_address"`
	Incentive   float64 `json:"incentive"`
	TaskCount   int64   `json:"task_count"`
	CardModel   string  `json:"card_model"`
	QoS         float64 `json:"qos"`
}

type GetNodeIncentiveData struct {
	Nodes []NodeIncentive `json:"nodes"`
}

type GetNodeIncentiveOutput struct {
	Data *GetNodeIncentiveData `json:"data"`
}

func GetNodeIncentive(_ *gin.Context, input *GetNodeIncentiveParams) (*GetNodeIncentiveOutput, error) {
	size := input.Size
	if size == 0 {
		size = 30
	}
	var start, end time.Time
	now := time.Now().UTC()
	if input.Period == UnitDay {
		duration := 24 * time.Hour
		end = now.Truncate(duration)
		start = end.Add(-duration)
	} else if input.Period == UnitWeek {
		duration := 7 * 24 * time.Hour
		end = now.Truncate(duration)
		start = end.Add(-duration)
	} else {
		year, month, _ := time.Now().UTC().Date()
		start = time.Date(year, month-1, 1, 0, 0, 0, 0, time.UTC)
		end = time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	}

	rows, err := config.GetDB().Model(&models.NodeIncentive{}).
		Select("node_address, SUM(incentive) as incentive, SUM(task_count) as task_count").
		Where("time >= ?", start).
		Where("time < ?", end).
		Group("node_address").
		Order("incentive DESC").
		Offset(0).
		Limit(size).
		Rows()

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	defer rows.Close()

	nodeIncentiveMap := make(map[string]NodeIncentive)
	var nodeAddresses []string
	for rows.Next() {
		var nodeAddress string
		var incentive float64
		var task_count int64

		if err := rows.Scan(&nodeAddress, &incentive, &task_count); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		nodeIncentive := NodeIncentive{
			NodeAddress: nodeAddress,
			Incentive:   incentive,
			TaskCount:   task_count,
		}
		nodeAddresses = append(nodeAddresses, nodeAddress)
		nodeIncentiveMap[nodeAddress] = nodeIncentive
	}

	var nodes []models.Node
	if err := config.GetDB().Model(&models.Node{}).Where("address IN (?)", nodeAddresses).Find(&nodes).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	for _, node := range nodes {
		if nodeIncentive, ok := nodeIncentiveMap[node.Address]; ok {
			nodeIncentive.CardModel = node.GPUName
			nodeIncentive.QoS = service.CalculateQosScore(node.QOSScore, service.GetMaxQosScore())

			nodeIncentiveMap[node.Address] = nodeIncentive
		}
	}

	var nodeIncentives []NodeIncentive
	for _, address := range nodeAddresses {
		if nodeIncentive, ok := nodeIncentiveMap[address]; ok {
			nodeIncentives = append(nodeIncentives, nodeIncentive)
		}
	}

	return &GetNodeIncentiveOutput{
		Data: &GetNodeIncentiveData{
			Nodes: nodeIncentives,
		},
	}, nil
}
