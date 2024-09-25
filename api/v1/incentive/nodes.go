package incentive

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	"github.com/gin-gonic/gin"
)

type TimeUnit string

const (
	UnitDay   TimeUnit = "Day"
	UnitWeek  TimeUnit = "Week"
	UnitMonth TimeUnit = "Month"
)

type GetNodeIncentiveParams struct {
	Period TimeUnit `query:"period" validate:"required" enum:"Day,Week,Month"`
	Size   int      `query:"size" default:"30"`
}

type NodeIncentive struct {
	NodeAddress string  `json:"node_address"`
	Incentive   float64 `json:"incentive"`
	TaskCount   int64   `json:"task_count"`
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
		Order("incentive").
		Offset(0).
		Limit(size).
		Rows()

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	defer rows.Close()

	var nodeIncentives []NodeIncentive
	for rows.Next() {
		var nodeAddress string
		var incentive float64
		var task_count int64

		if err := rows.Scan(&nodeAddress, &incentive, &task_count); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		nodeIncentives = append(nodeIncentives, NodeIncentive{
			NodeAddress: nodeAddress,
			Incentive:   incentive,
			TaskCount:   task_count,
		})
	}

	return &GetNodeIncentiveOutput{
		Data: &GetNodeIncentiveData{
			Nodes: nodeIncentives,
		},
	}, nil
}
