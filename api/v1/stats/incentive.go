package stats

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
)

type GetIncentiveLineChartParams struct {
	Period TimeUnit `query:"period" validate:"required" enum:"Day,Week,Month"`
	End    *int64   `query:"end"`
	Count  *int     `query:"count"`
}

type GetIncentiveLineChartData struct {
	Timestamps []int64   `json:"timestamps"`
	Incentives []float64 `json:"incentives"`
}

type GetIncentiveLineChartOutput struct {
	response.Response
	Data *GetIncentiveLineChartData `json:"data"`
}

type IncentiveResult struct {
	Index     int
	Incentive sql.NullFloat64
}

func GetIncentiveLineChart(_ *gin.Context, input *GetIncentiveLineChartParams) (*GetIncentiveLineChartOutput, error) {
	var times []time.Time
	now := time.Now().UTC()

	if input.Period == UnitDay {
		// 14 days
		duration := 24 * time.Hour
		var start time.Time
		count := 14
		if input.Count != nil {
			count = *input.Count
		}
		if input.End != nil {
			start = time.Unix(*input.End, 0).Truncate(duration).Add(-time.Duration(count) * duration)
		} else {
			start = now.Truncate(duration).Add(-time.Duration(count) * duration)
		}
		for i := 0; i < count+1; i++ {
			times = append(times, start)
			start = start.Add(duration)
		}
	} else if input.Period == UnitWeek {
		// 8 weeks
		duration := 7 * 24 * time.Hour
		var start time.Time
		count := 8
		if input.Count != nil {
			count = *input.Count
		}
		if input.End != nil {
			start = time.Unix(*input.End, 0).Truncate(duration).Add(-time.Duration(count) * duration)
		} else {
			start = now.Truncate(duration).Add(-time.Duration(count) * duration)
		}
		for i := 0; i < count+1; i++ {
			times = append(times, start)
			start = start.Add(duration)
		}
	} else {
		// 12 months
		end := now
		if input.End != nil {
			end = time.Unix(*input.End, 0)
		}
		count := 12
		if input.Count != nil {
			count = *input.Count
		}
		year, month, _ := end.Date()
		month -= time.Month(count)
		for i := 0; i < count+1; i++ {
			t := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
			times = append(times, t)
			month += 1
		}
	}

	// 构建时间范围条件
	args := make([]interface{}, 0)

	// 构建CASE WHEN语句
	caseWhen := "CASE "
	for i := 0; i < len(times)-1; i++ {
		caseWhen += "WHEN time >= ? AND time < ? THEN ? "
		args = append(args, times[i], times[i+1], i)
	}
	caseWhen += "END AS `index`"

	caseWhenExpr := config.GetDB().Dialector.Explain(caseWhen, args...)

	// 执行单个SQL查询
	var results []IncentiveResult
	query := config.GetDB().Model(&models.NodeIncentive{}).
		Select([]string{"SUM(incentive) as incentive", caseWhenExpr}).
		Where("time >= ? AND time < ?", times[0], times[len(times)-1]).
		Group("`index`").
		Order("`index`")

	if err := query.Scan(&results).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	// 处理结果
	timestamps := make([]int64, len(times)-1)
	incentives := make([]float64, len(times)-1)

	// 初始化所有时间段为0
	for i := 0; i < len(times)-1; i++ {
		timestamps[i] = times[i].Unix()
		incentives[i] = 0
	}

	// 填充有数据的时间段
	for _, result := range results {
		if result.Index >= 0 && result.Index < len(incentives) && result.Incentive.Valid {
			incentives[result.Index] = result.Incentive.Float64
		}
	}

	return &GetIncentiveLineChartOutput{
		Data: &GetIncentiveLineChartData{
			Timestamps: timestamps,
			Incentives: incentives,
		},
	}, nil
}
