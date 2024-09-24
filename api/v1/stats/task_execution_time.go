package stats

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type GetTaskExecutionTimeHistogramInput struct {
	TaskType TaskTypeString `query:"task_type" binding:"required,oneof=Image Text All"`
	Period   TimeUnit       `query:"period" binding:"required,oneof=Hour Day Week"`
}

type GetTaskExecutionTimeHistogramData struct {
	ExecutionTimes []int64 `json:"execution_times"`
	TaskCounts      []int64 `json:"task_count"`
}

type GetTaskExecutionTimeHistogramResponse struct {
	response.Response
	Data *GetTaskExecutionTimeHistogramData `json:"data"`
}

func GetTaskExecutionTimeHistogram(_ *gin.Context, input *GetTaskExecutionTimeHistogramInput) (*GetTaskExecutionTimeHistogramResponse, error) {
	now := time.Now().UTC()
	var start time.Time
	if input.Period == UnitHour {
		start = now.Truncate(time.Hour).Add(-time.Hour)
	} else if input.Period == UnitDay {
		start = now.Truncate(time.Hour).Add(-24 * time.Hour)
	} else {
		start = now.Truncate(time.Hour).Add(-7 * 24 * time.Hour)
	}

	var allTaskExecutionTimeCounts []models.TaskExecutionTimeCount
	stmt := config.GetDB().Model(&models.TaskExecutionTimeCount{}).Where("start >= ?", start)
	if input.TaskType == ImageTaskType {
		stmt = stmt.Where("task_type = ?", models.TaskTypeSD)
	} else if input.TaskType == TextTaskType {
		stmt = stmt.Where("task_type = ?", models.TaskTypeLLM)
	}
	stmt = stmt.Order("id")

	offset := 0
	for {
		var taskExecutionTimeCounts []models.TaskExecutionTimeCount
		if err := stmt.Offset(offset).Limit(1000).Find(&taskExecutionTimeCounts).Error; err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		allTaskExecutionTimeCounts = append(allTaskExecutionTimeCounts, taskExecutionTimeCounts...)
		if len(taskExecutionTimeCounts) < 1000 {
			break
		}
		offset += 1000
	}

	timeCounts := make(map[int64]int64)
	for _, taskExecutionTimeCount := range allTaskExecutionTimeCounts {
		timeCounts[taskExecutionTimeCount.Seconds] += taskExecutionTimeCount.Count
	}

	executionTimes := make([]int64, 0)
	for t := range timeCounts {
		executionTimes = append(executionTimes, t)
	}

	sort.Slice(executionTimes, func(i, j int) bool {
		return executionTimes[i] < executionTimes[j]
	})

	counts := make([]int64, 0)
	for _, t := range executionTimes {
		counts = append(counts, timeCounts[t])
	}

	return &GetTaskExecutionTimeHistogramResponse{
		Data: &GetTaskExecutionTimeHistogramData{
			ExecutionTimes: executionTimes,
			TaskCounts: counts,
		},
	}, nil
}
