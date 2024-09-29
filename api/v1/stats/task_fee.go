package stats

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"math"
	"time"

	"github.com/gin-gonic/gin"
)

type GetTaskFeeHistogramParams struct {
	TaskType TaskTypeString `query:"task_type" validate:"required" enum:"Image,Text,All"`
}

type GetTaskFeeHistogramData struct {
	TaskFees   []float64 `json:"task_fees"`
	TaskCounts []int64   `json:"task_counts"`
}

type GetTaskFeeHistogramOutput struct {
	Data *GetTaskFeeHistogramData `json:"data"`
}

func GetTaskFeeHistogram(_ *gin.Context, input *GetTaskFeeHistogramParams) (*GetTaskFeeHistogramOutput, error) {
	end := time.Now().UTC()
	start := end.Add(-time.Hour)

	type Fee struct {
		MaxFee float64
		MinFee float64
	}

	stmt := config.GetDB().Model(&models.InferenceTask{}).Where("created_at >= ?", start).Where("created_at < ?", end).Where("task_fee NOT NULL").Where("task_fee > ?", 0)
	if input.TaskType == ImageTaskType {
		stmt = stmt.Where("task_type = ?", models.TaskTypeSD)
	} else if input.TaskType == TextTaskType {
		stmt = stmt.Where("task_type = ?", models.TaskTypeLLM)
	}

	fee := Fee{}
	if err := stmt.Select("MAX(task_fee) as max_fee, MIN(task_fee) as min_fee").Scan(&fee).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	var taskFees []float64
	var taskCounts []int64

	if fee.MinFee < fee.MaxFee {

		binSize := math.Pow10(int(math.Floor(math.Log10(fee.MaxFee - fee.MinFee))))
		subquery := stmt.Select("id, CAST((task_fee / ?) AS UNSIGNED) AS f", binSize)
		rows, err := config.GetDB().Table("(?) AS t", subquery).Select("t.f as F, COUNT(t.id) AS count").Group("F").Order("F").Rows()
		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		defer rows.Close()

		fCounts := make(map[int64]int64)

		for rows.Next() {
			var F, count int64
			if err := rows.Scan(&F, &count); err != nil {
				return nil, response.NewExceptionResponse(err)
			}
			fCounts[F] = count
		}

		f := float64(int(fee.MinFee/binSize)) * binSize
		for i := 0; i < 10; i++ {
			taskFees = append(taskFees, f)
			F := int64(f / binSize)
			count, ok := fCounts[F]
			if !ok {
				count = 0
			}
			taskCounts = append(taskCounts, count)
			f += binSize
		}
	} else {
		var count int64
		if err := stmt.Select("COUNT(id) AS count").Scan(&count).Error; err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		taskCounts = append(taskCounts, count)
		for i := 0; i < 9; i++ {
			taskCounts = append(taskCounts, 0)
		}

		binSize := math.Pow10(int(math.Floor(math.Log10(fee.MinFee))))
		f := float64(int(fee.MinFee/binSize)) * binSize
		for i := 0; i < 10; i++ {
			taskFees = append(taskFees, f)
			f += binSize
		}
	}

	return &GetTaskFeeHistogramOutput{
		Data: &GetTaskFeeHistogramData{
			TaskFees:   taskFees,
			TaskCounts: taskCounts,
		},
	}, nil
}
