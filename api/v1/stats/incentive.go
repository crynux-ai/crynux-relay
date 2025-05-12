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

	var timestamps []int64
	var incentives []float64
	for i := 0; i < len(times)-1; i++ {
		start := times[i]
		end := times[i+1]
		timestamps = append(timestamps, start.Unix())

		var incentive sql.NullFloat64
		if err := config.GetDB().Model(&models.NodeIncentive{}).Select("SUM(incentive) AS incentive").Where("time >= ?", start).Where("time < ?", end).Scan(&incentive).Error; err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		if incentive.Valid {
			incentives = append(incentives, incentive.Float64)
		} else {
			incentives = append(incentives, 0)
		}
	}

	return &GetIncentiveLineChartOutput{
		Data: &GetIncentiveLineChartData{
			Timestamps: timestamps,
			Incentives: incentives,
		},
	}, nil
}
