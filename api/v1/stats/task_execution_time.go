package stats

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	"github.com/gin-gonic/gin"
)

type GetTaskExecutionTimeHistogramInput struct {
	TaskType TaskTypeString `query:"task_type" validate:"required" enum:"Image,Text,All"`
	Period   TimeUnit       `query:"period" validate:"required" enum:"Hour,Day,Week"`
}

type GetTaskExecutionTimeHistogramData struct {
	ExecutionTimes []int64 `json:"execution_times"`
	TaskCounts     []int64 `json:"task_count"`
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

	timeout := 300
	subQuery := config.GetDB().Model(&models.TaskExecutionTimeCount{}).Select("seconds, count").Where("start > ?", start).Where("seconds < ?", timeout)
	if input.TaskType == ImageTaskType {
		subQuery = subQuery.Where("task_type = ?", models.TaskTypeSD)
	} else if input.TaskType == TextTaskType {
		subQuery = subQuery.Where("task_type = ?", models.TaskTypeLLM)
	}

	rows, err := config.GetDB().Table("(?) as s", subQuery).Select("s.seconds AS seconds", "SUM(s.count) AS count").Group("seconds").Order("seconds").Rows()
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	defer rows.Close()

	seconds := make([]int64, 0)
	counts := make([]int64, 0)
	for rows.Next() {
		var second, count int64
		if err := rows.Scan(&second, &count); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		seconds = append(seconds, second)
		counts = append(counts, count)
	}

	return &GetTaskExecutionTimeHistogramResponse{
		Data: &GetTaskExecutionTimeHistogramData{
			ExecutionTimes: seconds,
			TaskCounts:     counts,
		},
	}, nil
}
