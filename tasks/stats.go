package tasks

import (
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var initStartTime time.Time = time.Date(2024, 7, 29, 0, 0, 0, 0, time.UTC)
var statsDuration time.Duration = time.Hour

func getTaskCounts(start, end time.Time) ([]*models.TaskCount, error) {
	var results []*models.TaskCount

	taskTypes := []models.ChainTaskType{models.TaskTypeSD, models.TaskTypeLLM}

	for _, taskType := range taskTypes {
		var successCount, abortedCount int64

		if err := config.GetDB().Model(&models.InferenceTask{}).Where("created_at >= ?", start).Where("created_at < ?", end).Where("task_type = ?", taskType).Where("status = ?", models.InferenceTaskAborted).Count(&abortedCount).Error; err != nil {
			log.Errorf("Stats: get %d type aborted task count error: %v", taskType, err)
			return nil, err
		}

		if err := config.GetDB().Model(&models.InferenceTask{}).Where("created_at >= ?", start).Where("created_at < ?", end).Where("task_type = ?", taskType).Where("status = ?", models.InferenceTaskResultsUploaded).Count(&successCount).Error; err != nil {
			log.Errorf("Stats: get %d type success task count error: %v", taskType, err)
			return nil, err
		}

		totalCount := successCount + abortedCount

		taskCount := models.TaskCount{
			Start:        start,
			End:          end,
			TaskType:     taskType,
			TotalCount:   totalCount,
			SuccessCount: successCount,
			AbortedCount: abortedCount,
		}

		results = append(results, &taskCount)
	}
	return results, nil
}

func statsTaskCount() error {
	taskCount := models.TaskCount{}
	if err := config.GetDB().Model(&models.TaskCount{}).Last(&taskCount).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Errorf("Stats: get last TaskCount error: %v", err)
			return err
		}
	}
	var start time.Time
	if taskCount.ID > 0 {
		start = taskCount.End
	} else {
		start = initStartTime
	}

	for {
		end := start.Add(statsDuration)
		if end.Sub(time.Now().UTC()) > 0 {
			break
		}
		taskCounts, err := getTaskCounts(start, end)
		if err != nil {
			return err
		}
		if err := config.GetDB().Create(taskCounts).Error; err != nil {
			log.Errorf("Stats: create TaskCount error: %v", err)
			return err
		}
		log.Infof("Stats: stats TaskCount success %s", end.Format(time.RFC3339))
		start = end
	}

	return nil
}

func StartStatsTaskCount() {
	for {
		statsTaskCount()
		time.Sleep(5 * time.Minute)
	}
}

func StartStatsTaskCountWithTerminateChannel(ch <-chan int) {
	for {
		select {
		case stop := <-ch:
			if stop == 1 {
				return
			} else {
				statsTaskCount()
			}
		default:
			statsTaskCount()
		}

		time.Sleep(5 * time.Minute)
	}
}

func getTaskExecutionTimeCount(start, end time.Time) ([]*models.TaskExecutionTimeCount, error) {
	var results []*models.TaskExecutionTimeCount

	taskTypes := []models.ChainTaskType{models.TaskTypeSD, models.TaskTypeLLM}

	for _, taskType := range taskTypes {
		rows, err := config.GetDB().Raw(
			"select s.T * 5 as T, count(s.id) as COUNT from (select t.id, CAST(TIMESTAMPDIFF(SECOND, t.created_at, t.updated_at) / 5 AS UNSIGNED) as T from inference_tasks t where t.created_at >= @start and t.created_at < @end and t.task_type = @taskType and t.status = @taskStatus) s group by T order by T",
			map[string]interface{}{"start": start, "end": end, "taskType": taskType, "taskStatus": models.InferenceTaskResultsUploaded}).Rows()
		if err != nil {
			log.Errorf("Stats: get %d type task execution time error: %v", taskType, err)
			return nil, err
		}
		defer rows.Close()
		var seconds, count int64
		for rows.Next() {
			rows.Scan(&seconds, &count)
			results = append(results, &models.TaskExecutionTimeCount{
				Start:    start,
				End:      end,
				TaskType: taskType,
				Seconds:  seconds,
				Count:    count,
			})
		}
	}
	return results, nil
}

func statsTaskExecutionTimeCount() error {
	taskExecutionTimeCount := models.TaskExecutionTimeCount{}

	if err := config.GetDB().Model(&models.TaskExecutionTimeCount{}).Last(&taskExecutionTimeCount).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Errorf("Stats: get last TaskExecutionTimeCount error: %v", err)
		}
	}
	var start time.Time
	if taskExecutionTimeCount.ID > 0 {
		start = taskExecutionTimeCount.End
	} else {
		start = initStartTime
	}

	for {
		end := start.Add(statsDuration)
		if end.Sub(time.Now().UTC()) > 0 {
			break
		}

		taskExecutionTimeCounts, err := getTaskExecutionTimeCount(start, end)
		if err != nil {
			return err
		}
		if err := config.GetDB().Create(taskExecutionTimeCounts).Error; err != nil {
			log.Errorf("Stats: create TaskExecutionTimeCount error: %v", err)
		}
		log.Infof("Stats: stats TaskExecutionTimeCount success: %s", end.Format(time.RFC3339))
		start = end
	}

	return nil
}

func StartStatsTaskExecutionTimeCount() {
	for {
		statsTaskExecutionTimeCount()
		time.Sleep(5 * time.Minute)
	}
}

func StartStatsTaskExecutionTimeCountWithTerminateChannel(ch <-chan int) {
	for {
		select {
		case stop := <-ch:
			if stop == 1 {
				return
			} else {
				statsTaskExecutionTimeCount()
			}
		default:
			statsTaskExecutionTimeCount()
		}
		time.Sleep(5 * time.Minute)
	}
}
