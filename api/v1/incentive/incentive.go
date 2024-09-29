package incentive

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	"github.com/gin-gonic/gin"
)

type GetIncentiveOutput struct {
	response.Response
	Data float64 `json:"data"`
}

func GetTotalIncentive(_ *gin.Context) (*GetIncentiveOutput, error) {
	var incentive float64
	if err := config.GetDB().Model(&models.NodeIncentive{}).Select("SUM(incentive) as incentive").Scan(&incentive).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetIncentiveOutput{Data: incentive}, nil
}

type GetIncentiveLineChartParams struct {
	Period TimeUnit `query:"period" validate:"required" enum:"Day,Week,Month"`
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
		start := now.Truncate(duration).Add(-13 * duration)
		for i := 0; i < 15; i++ {
			times = append(times, start)
			start = start.Add(duration)
		}
	} else if input.Period == UnitWeek {
		// 8 weeks
		duration := 7 * 24 * time.Hour
		start := now.Truncate(duration).Add(-7 * duration)
		for i := 0; i < 9; i++ {
			times = append(times, start)
			start = start.Add(duration)
		}
	} else {
		// 12 months
		year, month, _ := now.Date()
		year -= 1
		for i := 0; i < 13; i++ {
			t := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
			times = append(times, t)
			month += 1
		}
	}

	var timestamps []int64
	var incentives []float64
	for i := 0; i < len(times) - 1; i++ {
		start := times[i]
		end := times[i + 1]
		timestamps = append(timestamps, start.Unix())

		var incentive float64
		if err := config.GetDB().Model(&models.NodeIncentive{}).Select("SUM(incentive) AS incentive").Where("time >= ?", start).Where("time < ?", end).Scan(&incentive).Error; err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		incentives = append(incentives, incentive)
	}

	return &GetIncentiveLineChartOutput{
		Data: &GetIncentiveLineChartData{
			Timestamps: timestamps,
			Incentives: incentives,
		},
	}, nil
}
