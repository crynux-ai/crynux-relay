package incentive

import (
	"crynux_relay/api/v2/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
)

type GetNodeIncentiveParams struct {
	Period TimeUnit `query:"period" validate:"required" enum:"Day,Week,Month"`
	Size   int      `query:"size" default:"30"`
}

type NodeIncentive struct {
	NodeAddress       string  `json:"node_address"`
	Incentive         float64 `json:"incentive"`
	TaskCount         int64   `json:"task_count"`
	CardModel         string  `json:"card_model"`
	Staking           string  `json:"staking"`
	QOSScore          float64 `json:"qos_score"`
	StakingScore      float64 `json:"staking_score"`
	ProbWeight        float64 `json:"prob_weight"`
	SDTaskCount       int64   `json:"sd_task_count"`
	LLMTaskCount      int64   `json:"llm_task_count"`
	SDFTLoraTaskCount int64   `json:"sd_ft_lora_task_count"`
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
		Select("node_address, SUM(incentive) as incentive, SUM(task_count) as task_count, SUM(sd_task_count) as sd_task_count, SUM(llm_task_count) as llm_task_count, SUM(sd_ft_lora_task_count) as sd_ft_lora_task_count").
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
		var sd_task_count sql.NullInt64
		var llm_task_count sql.NullInt64
		var sd_ft_lora_task_count sql.NullInt64

		if err := rows.Scan(&nodeAddress, &incentive, &task_count, &sd_task_count, &llm_task_count, &sd_ft_lora_task_count); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		nodeIncentive := NodeIncentive{
			NodeAddress: nodeAddress,
			Incentive:   incentive,
			TaskCount:   task_count,
		}
		if sd_task_count.Valid {
			nodeIncentive.SDTaskCount = sd_task_count.Int64
		}
		if llm_task_count.Valid {
			nodeIncentive.LLMTaskCount = llm_task_count.Int64
		}
		if sd_ft_lora_task_count.Valid {
			nodeIncentive.SDFTLoraTaskCount = sd_ft_lora_task_count.Int64
		}
		nodeAddresses = append(nodeAddresses, nodeAddress)
		nodeIncentiveMap[nodeAddress] = nodeIncentive
	}

	var nodes []models.NetworkNodeData
	if err := config.GetDB().Model(&models.NetworkNodeData{}).Where("address IN (?)", nodeAddresses).Find(&nodes).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	for _, node := range nodes {
		
		if nodeIncentive, ok := nodeIncentiveMap[node.Address]; ok {
			stakingProb, qosProb, prob := service.CalculateSelectingProb(&node.Staking.Int, service.GetMaxStaking(), node.QoS, service.GetMaxQosScore())
			nodeIncentive.CardModel = node.CardModel
			nodeIncentive.QOSScore = qosProb
			nodeIncentive.Staking = node.Staking.String()
			nodeIncentive.StakingScore = stakingProb
			nodeIncentive.ProbWeight = prob

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
