package stats

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type GetTaskSuccessRateLineChartParams struct {
	TaskType TaskTypeString `query:"task_type" validate:"required" enum:"Image,Text,All"`
	Period   TimeUnit       `query:"period" validate:"required" enum:"Hour,Day,Week"`
}

type GetTaskSuccessRateLineChartData struct {
	Timestamps   []int64   `json:"timestamps"`
	SuccessRates []float64 `json:"success_rate"`
}

type GetTaskSuccessRateLineChartOutput struct {
	response.Response
	Data *GetTaskSuccessRateLineChartData `json:"data"`
}

func GetTaskSuccessRateLineChart(_ *gin.Context, input *GetTaskCountLineChartParams) (*GetTaskSuccessRateLineChartOutput, error) {
	timestampSuccessCount := make(map[int64]int64)
	timestampTotalCount := make(map[int64]int64)

	now := time.Now().UTC()
	var start time.Time
	var duration time.Duration
	if input.Period == UnitHour {
		duration = time.Hour
		start = now.Truncate(duration).Add(-24 * duration)
	} else if input.Period == UnitDay {
		duration = 24 * time.Hour
		start = now.Add(duration - 1).Truncate(duration).Add(-15 * duration)
	} else {
		duration = 7 * 24 * time.Hour
		start = now.Add(duration - 1).Truncate(duration).Add(-8 * duration)
	}

	var allTaskCounts []models.TaskCount
	stmt := config.GetDB().Model(&models.TaskCount{}).Where("start >= ?", start)
	if input.TaskType == ImageTaskType {
		stmt = stmt.Where("task_type = ?", models.TaskTypeSD)
	} else if input.TaskType == TextTaskType {
		stmt = stmt.Where("task_type = ?", models.TaskTypeLLM)
	}
	stmt = stmt.Order("id")

	offset := 0
	for {
		var taskCounts []models.TaskCount
		if err := stmt.Offset(offset).Limit(1000).Find(&taskCounts).Error; err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		allTaskCounts = append(allTaskCounts, taskCounts...)
		if len(taskCounts) < 1000 {
			break
		}
		offset += 1000
	}

	for _, taskCount := range allTaskCounts {
		timestamp := taskCount.Start.Truncate(duration).Unix()
		timestampTotalCount[timestamp] += taskCount.TotalCount
		timestampSuccessCount[timestamp] += taskCount.SuccessCount
	}

	timestamps := make([]int64, 0)
	for timestamp := range timestampTotalCount {
		timestamps = append(timestamps, timestamp)
	}

	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i] < timestamps[j]
	})

	successRates := make([]float64, 0)
	for _, timestamp := range timestamps {
		rate := float64(timestampSuccessCount[timestamp]) / float64(timestampTotalCount[timestamp])
		successRates = append(successRates, rate)
	}

	return &GetTaskSuccessRateLineChartOutput{
		Data: &GetTaskSuccessRateLineChartData{
			Timestamps:   timestamps,
			SuccessRates: successRates,
		},
	}, nil
}
