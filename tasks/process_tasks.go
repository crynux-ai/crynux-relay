package tasks

import (
	"context"
	"crynux_relay/blockchain"
	"crynux_relay/blockchain/bindings"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

func getTask(ctx context.Context, taskIDCommitment string) (*models.InferenceTask, error) {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	task := models.InferenceTask{TaskIDCommitment: taskIDCommitment}
	if err := config.GetDB().WithContext(dbCtx).Model(&task).Where(&task).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func doWaitTaskStatus(ctx context.Context, taskIDCommitment string, status models.TaskStatus) error {
	for {
		task, err := getTask(ctx, taskIDCommitment)
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
	go func()  {
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

func processOneTask(ctx context.Context, task *models.InferenceTask) error {
	taskIDCommitmentBytes, err := utils.HexStrToCommitment(task.TaskIDCommitment)
	if err != nil {
		return err
	}
	// report task params is uploaded to blochchain
	if task.Status == models.InferenceTaskCreated {

		receipt, err := func() (*types.Receipt, error) {
			callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()
			txHash, err := blockchain.ReportTaskParamsUploaded(callCtx, *taskIDCommitmentBytes)
			if err != nil {
				return nil, err
			}

			receipt, err := blockchain.WaitTxReceipt(callCtx, common.HexToHash(txHash))
			return receipt, err
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

		err = func() error {
			dbCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			task.Status = models.InferenceTaskParamsUploaded
			task.StartTime = time.Now().UTC()
			if err := config.GetDB().WithContext(dbCtx).Save(task).Error; err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	// wait task has been validated
	needResult := false
	if task.Status == models.InferenceTaskParamsUploaded {
		for task.ValidatedTime.IsZero() {
			time.Sleep(time.Second)

			chainTask, err := func() (*bindings.VSSTaskTaskInfo, error) {
				callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				return blockchain.GetTaskByCommitment(callCtx, *taskIDCommitmentBytes)
			}()
			if err != nil {
				return err
			}

			changed := false
			chainTaskStatus := models.ChainTaskStatus(chainTask.Status)
			abortReason := models.TaskAbortReason(chainTask.AbortReason)
			taskError := models.TaskError(chainTask.Error)
			scoreReadyTimestamp := chainTask.ScoreReadyTimestamp.Int64()

			if scoreReadyTimestamp > 0 {
				task.ScoreReadyTime = time.Unix(scoreReadyTimestamp, 0).UTC()
				changed = true
			}
			if abortReason != models.TaskAbortReasonNone {
				task.AbortReason = abortReason
				changed = true
			}
			if taskError != models.TaskErrorNone {
				task.TaskError = taskError
				changed = true
			}

			if chainTaskStatus == models.ChainTaskValidated || chainTaskStatus == models.ChainTaskGroupValidated {
				task.ValidatedTime = time.Now().UTC()
				needResult = true
				changed = true
			} else if chainTaskStatus == models.ChainTaskEndAborted {
				task.Status = models.InferenceTaskEndAborted
				task.ValidatedTime = time.Now().UTC()
				changed = true
			} else if chainTaskStatus == models.ChainTaskEndInvalidated {
				task.Status = models.InferenceTaskEndInvalidated
				task.ValidatedTime = time.Now().UTC()
				changed = true
			} else if chainTaskStatus == models.ChainTaskEndGroupRefund {
				task.Status = models.InferenceTaskEndGroupRefund
				task.ValidatedTime = time.Now().UTC()
				changed = true
			}

			if changed {
				err := func() error {
					dbCtx, cancel := context.WithTimeout(ctx, time.Second)
					defer cancel()
					if err := config.GetDB().WithContext(dbCtx).Save(task).Error; err != nil {
						return err
					}
					return nil
				}()
				if err != nil {
					return err
				}
			}
		}
	}

	// report task result is uploaded to blockchain
	if needResult {
		// wait task result is ready
		err := func () error {
			timeCtx, cancel := context.WithTimeout(ctx, 5 * time.Minute)
			defer cancel()
			return waitTaskStatus(timeCtx, task.TaskIDCommitment, models.InferenceTaskResultsReady)
		}()
		if err != nil {
			return err
		}
		// task result is uploaded
		receipt, err := func() (*types.Receipt, error) {
			callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()
			txHash, err := blockchain.ReportTaskResultUploaded(callCtx, *taskIDCommitmentBytes)
			if err != nil {
				return nil, err
			}

			receipt, err := blockchain.WaitTxReceipt(callCtx, common.HexToHash(txHash))
			return receipt, err
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

		err = func() error {
			dbCtx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			task.Status = models.InferenceTaskEndSuccess
			task.ResultUploadedTime = time.Now().UTC()
			if err := config.GetDB().WithContext(dbCtx).Save(task).Error; err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func ProcessTasks(ctx context.Context) {
	limit := 100
	lastID := uint(0)

	interval := 1

	for {
		tasks, err := func (ctx context.Context) ([]models.InferenceTask, error) {
			var tasks []models.InferenceTask
	
			dbCtx, cancel := context.WithTimeout(ctx, 3 * time.Second)
			defer cancel()
			err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
				Where("status != ?", models.InferenceTaskEndAborted).
				Where("status != ?", models.InferenceTaskEndInvalidated).
				Where("status != ?", models.InferenceTaskEndSuccess).
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
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}

		if len(tasks) > 0 {
			lastID = tasks[len(tasks) - 1].ID
			
			for _, task := range tasks {
				go func (ctx context.Context, task models.InferenceTask)  {
					log.Infof("ProcessTasks: start processing task %s", task.TaskIDCommitment)
					var ctx1 context.Context
					var cancel context.CancelFunc
					if task.StartTime.IsZero() {
						ctx1, cancel = context.WithTimeout(ctx, 10 * time.Minute)
					} else {
						deadline := task.StartTime.Add(10 * time.Minute)
						ctx1, cancel = context.WithDeadline(ctx, deadline)
					}
					defer cancel()

					for {
						c := make(chan error, 1)
						go func ()  {
							c <- processOneTask(ctx1, &task)
						}()
						
						select {
						case err := <- c:
							if err != nil {
								log.Errorf("ProcessTasks: process task %s error %v, retry", task.TaskIDCommitment, err)
							} else {
								log.Infof("ProcessTasks: process task %s successfully", task.TaskIDCommitment)
								return
							}
						case <- ctx1.Done():
							err := ctx1.Err()
							log.Errorf("ProcessTasks: process task %s timeout %v, finish", task.TaskIDCommitment, err)
							// set task status to aborted to avoid processing it again
							if err == context.DeadlineExceeded {
								task.Status = models.InferenceTaskEndAborted
								err = func() error {
									dbCtx, cancel := context.WithTimeout(context.Background(), time.Second)
									defer cancel()
									if err := config.GetDB().WithContext(dbCtx).Save(&task).Error; err != nil {
										return err
									}
									return nil
								}()
								if err != nil {
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
