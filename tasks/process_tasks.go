package tasks

import (
	"context"
	"crynux_relay/blockchain"
	"crynux_relay/blockchain/bindings"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"database/sql"
	"errors"
	mrand "math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

func getChainTask(ctx context.Context, taskIDCommitmentBytes [32]byte) (*bindings.VSSTaskTaskInfo, error) {
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return blockchain.GetTaskByCommitment(callCtx, taskIDCommitmentBytes)
}

func reportTaskParamsUploaded(ctx context.Context, task *models.InferenceTask) error {
	taskIDCommitmentBytes, err := utils.HexStrToCommitment(task.TaskIDCommitment)
	if err != nil {
		return nil
	}

	txHash, err := func() (string, error) {
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return blockchain.ReportTaskParamsUploaded(callCtx, *taskIDCommitmentBytes)
	}()
	if err != nil {
		return err
	}

	receipt, err := func() (*types.Receipt, error) {
		callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
		return blockchain.WaitTxReceipt(callCtx, common.HexToHash(txHash))
	}()
	if err != nil {
		return err
	}
	if receipt.Status == 0 {
		errMsg, err := blockchain.GetErrorMessageFromReceipt(ctx, receipt)
		if err != nil {
			return err
		}
		log.Errorf("ProcessTasks: %s reportTaskParamsUploaded failed: %s", task.TaskIDCommitment, errMsg)
		return errors.New(errMsg)
	}
	return nil
}

func reportTaskResultUploaded(ctx context.Context, task *models.InferenceTask) error {
	taskIDCommitmentBytes, err := utils.HexStrToCommitment(task.TaskIDCommitment)
	if err != nil {
		return nil
	}

	txHash, err := func() (string, error) {
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return blockchain.ReportTaskResultUploaded(callCtx, *taskIDCommitmentBytes)
	}()
	if err != nil {
		return err
	}

	receipt, err := func() (*types.Receipt, error) {
		callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
		return blockchain.WaitTxReceipt(callCtx, common.HexToHash(txHash))
	}()
	if err != nil {
		return err
	}

	if receipt.Status == 0 {
		errMsg, err := blockchain.GetErrorMessageFromReceipt(ctx, receipt)
		if err != nil {
			return err
		}
		log.Errorf("ProcessTasks: %s reportTaskResultUploaded failed: %s", task.TaskIDCommitment, errMsg)
		return errors.New(errMsg)
	}
	return nil
}

func doWaitTaskStatus(ctx context.Context, taskIDCommitment string, status models.TaskStatus) error {
	for {
		task, err := models.GetTaskByIDCommitment(ctx, config.GetDB(), taskIDCommitment)
		if err != nil {
			return err
		}
		if task.Status == status {
			return nil
		}
		time.Sleep(time.Second)
	}
}

func waitTaskStatus(ctx context.Context, taskIDCommitment string, status models.TaskStatus) error {
	c := make(chan error, 1)
	go func() {
		c <- doWaitTaskStatus(ctx, taskIDCommitment, status)
	}()
	select {
	case err := <-c:
		return err
	case <-ctx.Done():
		log.Errorf("ProcessTasks: check task %s status %d timed out", taskIDCommitment, status)
		return ctx.Err()
	}
}

func syncTask(ctx context.Context, task *models.InferenceTask) (*bindings.VSSTaskTaskInfo, error) {
	taskIDCommitmentBytes, err := utils.HexStrToCommitment(task.TaskIDCommitment)
	if err != nil {
		return nil, err
	}

	chainTask, err := getChainTask(ctx, *taskIDCommitmentBytes)
	if err != nil {
		return nil, err
	}

	changed := false
	newTask := make(map[string]interface{})
	chainTaskStatus := models.TaskStatus(chainTask.Status)
	abortReason := models.TaskAbortReason(chainTask.AbortReason)
	taskError := models.TaskError(chainTask.Error)
	startTimestamp := chainTask.StartTimestamp.Int64()
	scoreReadyTimestamp := chainTask.ScoreReadyTimestamp.Int64()

	if startTimestamp > 0 && !task.StartTime.Valid {
		newTask["start_time"] = sql.NullTime{
			Time:  time.Unix(startTimestamp, 0).UTC(),
			Valid: true,
		}
		changed = true
	}
	if scoreReadyTimestamp > 0 && !task.ScoreReadyTime.Valid {
		newTask["score_ready_time"] = sql.NullTime{
			Time:  time.Unix(scoreReadyTimestamp, 0).UTC(),
			Valid: true,
		}
		changed = true
	}
	if abortReason != task.AbortReason {
		newTask["abort_reason"] = abortReason
		changed = true
	}
	if taskError != task.TaskError {
		newTask["task_error"] = taskError
		changed = true
	}
	if task.Status != chainTaskStatus {
		task.Status = chainTaskStatus
		changed = true
	}
	if chainTaskStatus == models.TaskValidated || chainTaskStatus == models.TaskGroupValidated {
		if !task.ValidatedTime.Valid {
			newTask["validated_time"] = sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			}
			changed = true
		}
	} else if chainTaskStatus == models.TaskEndAborted {
		if !task.ValidatedTime.Valid {
			newTask["validated_time"] = sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			}
			changed = true
		}
	} else if chainTaskStatus == models.TaskEndInvalidated {
		if !task.ValidatedTime.Valid {
			newTask["validated_time"] = sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			}
			changed = true
		}
	} else if chainTaskStatus == models.TaskEndGroupRefund {
		if !task.ValidatedTime.Valid {
			newTask["validated_time"] = sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			}
			changed = true
		}
	}

	if changed {
		if err := task.Update(ctx, config.GetDB(), newTask); err != nil {
			return nil, err
		}
	}
	return chainTask, nil
}

func processOneTask(ctx context.Context, task *models.InferenceTask) error {
	// sync task from blockchain first
	_, err := syncTask(ctx, task)
	if err != nil {
		return err
	}

	// report task params is uploaded to blochchain
	if task.Status == models.TaskStarted {
		if err := reportTaskParamsUploaded(ctx, task); err != nil {
			return err
		}

		newTask := map[string]interface{}{
			"status": models.TaskParametersUploaded,
		}

		if err := task.Update(ctx, config.GetDB(), newTask); err != nil {
			return err
		}
		log.Infof("ProcessTasks: report task %s params uploaded", task.TaskIDCommitment)
	}

	// wait task has been validated or end
	needResult := false
	for {
		chainTask, err := syncTask(ctx, task)
		if err != nil {
			return err
		}
		chainTaskStatus := models.TaskStatus(chainTask.Status)
		needResult = (chainTaskStatus == models.TaskValidated || chainTaskStatus == models.TaskGroupValidated)
		if task.Status != models.TaskParametersUploaded || task.ValidatedTime.Valid {
			break
		}
		time.Sleep(time.Second)
	}

	// report task result is uploaded to blockchain
	if needResult {
		// wait task result is ready
		log.Infof("ProcessTasks: task %s is validated", task.TaskIDCommitment)
		err := func() error {
			timeCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			return waitTaskStatus(timeCtx, task.TaskIDCommitment, models.TaskEndSuccess)
		}()
		if err != nil {
			return err
		}
		log.Infof("ProcessTasks: task %s result is uploaded", task.TaskIDCommitment)
		// task result is uploaded
		if err := reportTaskResultUploaded(ctx, task); err != nil {
			return err
		}

		newTask := map[string]interface{}{
			"status": models.TaskEndSuccess,
			"result_uploaded_time": sql.NullTime{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		}

		if err := task.Update(ctx, config.GetDB(), newTask); err != nil {
			return err
		}
		log.Infof("ProcessTasks: report task %s result is uploaded", task.TaskIDCommitment)
	} else {
		log.Infof("ProcessTasks: task %s finished with status: %d", task.TaskIDCommitment, task.Status)
	}
	return nil
}

func ProcessTasks(ctx context.Context) {
	limit := 100
	lastID := uint(0)

	for {
		tasks, err := func(ctx context.Context) ([]models.InferenceTask, error) {
			var tasks []models.InferenceTask

			dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
				Where("status != ?", models.TaskEndAborted).
				Where("status != ?", models.TaskEndInvalidated).
				Where("status != ?", models.TaskEndSuccess).
				Where("id > ?", lastID).
				Find(&tasks).
				Order("id ASC").
				Limit(limit).
				Error
			if err != nil {
				return nil, err
			}
			return tasks, nil
		}(ctx)
		if err != nil {
			log.Errorf("ProcessTasks: cannot get unprocessed tasks: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if len(tasks) > 0 {
			lastID = tasks[len(tasks)-1].ID

			for _, task := range tasks {
				go func(ctx context.Context, task models.InferenceTask) {
					log.Infof("ProcessTasks: start processing task %s", task.TaskIDCommitment)
					var ctx1 context.Context
					var cancel context.CancelFunc
					if !task.StartTime.Valid {
						duration := 3 * time.Minute
						ctx1, cancel = context.WithTimeout(ctx, duration)
					} else {
						duration := time.Duration(task.Timeout) * time.Minute + 3 * time.Minute
						deadline := task.StartTime.Time.Add(duration)
						ctx1, cancel = context.WithDeadline(ctx, deadline)
					}
					defer cancel()

					for {
						c := make(chan error, 1)
						go func() {
							c <- processOneTask(ctx1, &task)
						}()

						select {
						case err := <-c:
							if err != nil {
								log.Errorf("ProcessTasks: process task %s error %v, retry", task.TaskIDCommitment, err)
								duration := time.Duration((mrand.Float64()*2 + 1) * 1000)
								time.Sleep(duration * time.Millisecond)
							} else {
								log.Infof("ProcessTasks: process task %s successfully", task.TaskIDCommitment)
								return
							}
						case <-ctx1.Done():
							err := ctx1.Err()
							log.Errorf("ProcessTasks: process task %s timeout %v, finish", task.TaskIDCommitment, err)
							// set task status to aborted to avoid processing it again
							if err == context.DeadlineExceeded {
								newTask := map[string]interface{}{"status": models.TaskEndAborted}
								if err := task.Update(ctx, config.GetDB(), newTask); err != nil {
									log.Errorf("ProcessTasks: save task %s error %v", task.TaskIDCommitment, err)
								}
							}
							return
						}
					}
				}(ctx, task)
			}
		}

		time.Sleep(time.Second)
	}
}
