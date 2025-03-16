package tasks

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var initStartTime time.Time = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
var statsDuration time.Duration = time.Hour

func getTaskCounts(ctx context.Context, start, end time.Time) ([]*models.TaskCount, error) {
	var results []*models.TaskCount

	taskTypes := []models.TaskType{models.TaskTypeSD, models.TaskTypeLLM, models.TaskTypeSDFTLora}

	for _, taskType := range taskTypes {
		var successCount, abortedCount int64

		err := func() error {
			dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
				Where("created_at >= ?", start).Where("created_at < ?", end).
				Where("task_type = ?", taskType).
				Where("(status = ? OR status = ?)", models.TaskEndAborted, models.TaskEndInvalidated).
				Count(&abortedCount).Error; err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			log.Errorf("Stats: get %d type aborted task count error: %v", taskType, err)
			return nil, err
		}

		err = func() error {
			dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
				Where("created_at >= ?", start).Where("created_at < ?", end).
				Where("task_type = ?", taskType).
				Where("(status = ? OR status = ?)", models.TaskEndSuccess, models.TaskEndGroupRefund).
				Count(&successCount).Error; err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
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

func statsTaskCount(ctx context.Context) error {
	now := time.Now().UTC()
	taskCount := models.TaskCount{}
	err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&models.TaskCount{}).Last(&taskCount).Error
	}()
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("Stats: get last TaskCount error: %v", err)
		return err
	}

	var start time.Time
	if taskCount.ID > 0 {
		start = taskCount.End
	} else {
		start = initStartTime
	}

	for {
		end := start.Add(statsDuration)
		if end.Sub(now) > 0 {
			break
		}
		taskCounts, err := getTaskCounts(ctx, start, end)
		if err != nil {
			return err
		}
		if len(taskCounts) > 0 {
			err := func() error {
				dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				return config.GetDB().WithContext(dbCtx).Create(taskCounts).Error
			}()
			if err != nil {
				log.Errorf("Stats: create TaskCount error: %v", err)
				return err
			}
		}
		log.Infof("Stats: stats TaskCount success %s", end.Format(time.RFC3339))
		start = end
	}

	return nil
}

func StartStatsTaskCount(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			ticker.Stop()
			log.Infof("Stats: stop counting task count due to %v", err)
			return
		case <-ticker.C:
			func() {
				ctx1, cancel := context.WithTimeout(ctx, 5*time.Minute)
				defer cancel()
				if err := statsTaskCount(ctx1); err != nil {
					log.Errorf("Stats: stats task count error %v", err)
				}
			}()
		}
	}
}

func getTaskExecutionTimeCount(ctx context.Context, start, end time.Time) ([]*models.TaskExecutionTimeCount, error) {
	var results []*models.TaskExecutionTimeCount

	taskTypes := []models.TaskType{models.TaskTypeSD, models.TaskTypeLLM, models.TaskTypeSDFTLora}
	binSize := 5
	for _, taskType := range taskTypes {
		rows, err := func() (*sql.Rows, error) {
			dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			subQuery := config.GetDB().Table("inference_tasks").
				Select("id, CAST(TIMESTAMPDIFF(SECOND, start_time, score_ready_time) / ? AS SIGNED) AS time", binSize).
				Where("created_at >= ?", start).Where("created_at < ?", end).
				Where("task_type = ?", taskType).
				Where("score_ready_time IS NOT NULL")
			return config.GetDB().WithContext(dbCtx).
				Table("(?) AS s", subQuery).
				Select("s.time * ? as T, COUNT(s.id) AS count", binSize).
				Where("s.time >= 0").
				Group("T").Order("T").Rows()
		}()

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

func statsTaskExecutionTimeCount(ctx context.Context) error {
	now := time.Now().UTC()

	taskExecutionTimeCount := models.TaskExecutionTimeCount{}
	err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&models.TaskExecutionTimeCount{}).Last(&taskExecutionTimeCount).Error
	}()
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("Stats: get last TaskExecutionTimeCount error: %v", err)
		return err
	}
	var start time.Time
	if taskExecutionTimeCount.ID > 0 {
		start = taskExecutionTimeCount.End
	} else {
		start = initStartTime
	}

	for {
		end := start.Add(statsDuration)
		if end.Sub(now) > 0 {
			break
		}

		taskExecutionTimeCounts, err := getTaskExecutionTimeCount(ctx, start, end)
		if err != nil {
			return err
		}
		if len(taskExecutionTimeCounts) > 0 {
			err := func() error {
				dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				return config.GetDB().WithContext(dbCtx).Create(taskExecutionTimeCounts).Error
			}()
			if err != nil {
				log.Errorf("Stats: create TaskExecutionTimeCount error: %v", err)
				return err
			}
		}
		log.Infof("Stats: stats TaskExecutionTimeCount success: %s", end.Format(time.RFC3339))
		start = end
	}

	return nil
}

func StartStatsTaskExecutionTimeCount(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			log.Errorf("Stats: stop counting task execution time count due to %v", err)
			ticker.Stop()
		case <-ticker.C:
			func() {
				ctx1, cancel := context.WithTimeout(ctx, 5*time.Minute)
				defer cancel()
				if err := statsTaskExecutionTimeCount(ctx1); err != nil {
					log.Errorf("Stats: stats task execution time count error %v", err)
				}
			}()
		}
	}
}

func getTaskUploadResultTimeCount(ctx context.Context, start, end time.Time) ([]*models.TaskUploadResultTimeCount, error) {
	var results []*models.TaskUploadResultTimeCount

	taskTypes := []models.TaskType{models.TaskTypeSD, models.TaskTypeLLM, models.TaskTypeSDFTLora}
	binSize := 5
	for _, taskType := range taskTypes {
		rows, err := func() (*sql.Rows, error) {
			dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			subQuery := config.GetDB().Table("inference_tasks").
				Select("id, CAST(TIMESTAMPDIFF(SECOND, validated_time, result_uploaded_time) / ? AS SIGNED) AS time", binSize).
				Where("created_at >= ?", start).Where("created_at < ?", end).
				Where("task_type = ?", taskType).
				Where("result_uploaded_time IS NOT NULL")
			return config.GetDB().WithContext(dbCtx).
				Table("(?) AS s", subQuery).
				Select("s.time * ? as T, COUNT(s.id) AS count", binSize).
				Where("s.time >= 0").
				Group("T").Order("T").Rows()
		}()
		if err != nil {
			log.Errorf("Stats: get %d type task result upload time error: %v", taskType, err)
			return nil, err
		}
		defer rows.Close()
		var seconds, count int64
		for rows.Next() {
			rows.Scan(&seconds, &count)
			results = append(results, &models.TaskUploadResultTimeCount{
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

func statsTaskUploadResultTimeCount(ctx context.Context) error {
	now := time.Now().UTC()
	taskUploadResultTimeCount := models.TaskUploadResultTimeCount{}

	err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&models.TaskUploadResultTimeCount{}).Last(&taskUploadResultTimeCount).Error
	}()
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("Stats: get last TaskUploadResultTime error: %v", err)
		return err
	}
	var start time.Time
	if taskUploadResultTimeCount.ID > 0 {
		start = taskUploadResultTimeCount.End
	} else {
		start = initStartTime
	}

	for {
		end := start.Add(statsDuration)
		if end.Sub(now) > 0 {
			break
		}

		taskUploadResultTimeCounts, err := getTaskUploadResultTimeCount(ctx, start, end)
		if err != nil {
			return err
		}
		if len(taskUploadResultTimeCounts) > 0 {
			err := func() error {
				dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				return config.GetDB().WithContext(dbCtx).Create(taskUploadResultTimeCounts).Error
			}()
			if err != nil {
				log.Errorf("Stats: create TaskUploadResultTimeCount error: %v", err)
				return err
			}
		}
		log.Infof("Stats: stats TaskUploadResultTimeCount success: %s", end.Format(time.RFC3339))
		start = end
	}

	return nil
}

func StartStatsTaskUploadResultTimeCount(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			log.Errorf("Stats: stop counting task upload result time count due to %v", err)
			ticker.Stop()
		case <-ticker.C:
			func() {
				ctx1, cancel := context.WithTimeout(ctx, 5*time.Minute)
				defer cancel()
				if err := statsTaskUploadResultTimeCount(ctx1); err != nil {
					log.Errorf("Stats: stats task upload result time count error %v", err)
				}
			}()
		}
	}
}

func getTaskWaitingTimeCount(ctx context.Context, start, end time.Time) ([]*models.TaskWaitingTimeCount, error) {
	var results []*models.TaskWaitingTimeCount

	taskTypes := []models.TaskType{models.TaskTypeSD, models.TaskTypeLLM, models.TaskTypeSDFTLora}
	binSize := 5
	for _, taskType := range taskTypes {
		rows, err := func() (*sql.Rows, error) {
			dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			subQuery := config.GetDB().Table("inference_tasks").
				Select("id, CAST(TIMESTAMPDIFF(SECOND, create_time, start_time) / ? AS SIGNED) AS time", binSize).
				Where("created_at >= ?", start).Where("created_at < ?", end).
				Where("task_type = ?", taskType).
				Where("start_time IS NOT NULL")
			return config.GetDB().WithContext(dbCtx).Table("(?) AS s", subQuery).
				Select("s.time * ? as T, COUNT(s.id) AS count", binSize).
				Where("s.time >= 0").
				Group("T").Order("T").Rows()
		}()
		if err != nil {
			log.Errorf("Stats: get %d type task result upload time error: %v", taskType, err)
			return nil, err
		}
		defer rows.Close()
		var seconds, count int64
		for rows.Next() {
			rows.Scan(&seconds, &count)
			results = append(results, &models.TaskWaitingTimeCount{
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

func statsTaskWaitingTimeCount(ctx context.Context) error {
	now := time.Now().UTC()

	taskWaitingTimeCount := models.TaskWaitingTimeCount{}
	err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&models.TaskWaitingTimeCount{}).Last(&taskWaitingTimeCount).Error
	}()
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("Stats: get last TaskWaitingTimeCount error: %v", err)
		return err
	}
	var start time.Time
	if taskWaitingTimeCount.ID > 0 {
		start = taskWaitingTimeCount.End
	} else {
		start = initStartTime
	}

	for {
		end := start.Add(statsDuration)
		if end.Sub(now) > 0 {
			break
		}

		taskWaitingTimeCounts, err := getTaskWaitingTimeCount(ctx, start, end)
		if err != nil {
			return err
		}
		if len(taskWaitingTimeCounts) > 0 {
			err := func() error {
				dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				return config.GetDB().WithContext(dbCtx).Create(taskWaitingTimeCounts).Error
			}()
			if err != nil {
				log.Errorf("Stats: create TaskWaitingTimeCount error: %v", err)
				return err
			}
		}
		log.Infof("Stats: stats TaskWaitingTimeCount success: %s", end.Format(time.RFC3339))
		start = end
	}

	return nil
}

func StartStatsTaskWaitingTimeCount(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			log.Errorf("Stats: stop counting task waiting time count due to %v", err)
			ticker.Stop()
		case <-ticker.C:
			func() {
				ctx1, cancel := context.WithTimeout(ctx, 5*time.Minute)
				defer cancel()
				if err := statsTaskWaitingTimeCount(ctx1); err != nil {
					log.Errorf("Stats: stats task waiting time count error %v", err)
				}
			}()
		}
	}
}
