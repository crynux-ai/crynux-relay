package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"
	"errors"
	"math/big"
	"time"

	"gorm.io/gorm"
)

func CreateTask(ctx context.Context, db *gorm.DB, task *models.InferenceTask) error {
	appConfig := config.GetConfig()

	return db.Transaction(func(tx *gorm.DB) error {
		if err := task.Save(ctx, tx); err != nil {
			return err
		}
		return Transfer(ctx, tx, task.Creator, appConfig.Blockchain.Account.Address, &task.TaskFee.Int)
	})
}

func SetTaskStatusStarted(ctx context.Context, db *gorm.DB, task *models.InferenceTask, node *models.Node) error {
	// start inference task
	err := db.Transaction(func(tx *gorm.DB) error {
		dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := task.Update(dbCtx, tx, &models.InferenceTask{
			SelectedNode: node.Address,
			StartTime:    sql.NullTime{Time: time.Now(), Valid: true},
			Status:       models.TaskStarted,
		}); err != nil {
			return err
		}

		if err := nodeStartTask(dbCtx, tx, node, task.TaskIDCommitment, task.ModelIDs); err != nil {
			return err
		}
		return emitEvent(ctx, tx, &models.TaskStartedEvent{
			TaskIDCommitment: task.TaskIDCommitment,
			SelectedNode:     node.Address,
		})
	})
	if err != nil {
		return err
	}

	// start download tasks
	localModelSet := make(map[string]models.NodeModel)
	for _, model := range node.Models {
		localModelSet[model.ModelID] = model
	}

	for _, modelID := range task.ModelIDs {
		download := false
		if _, ok := localModelSet[modelID]; !ok {
			emitEvent(ctx, db, &models.DownloadModelEvent{
				NodeAddress: node.Address,
				ModelID:     modelID,
				TaskType:    task.TaskType,
			})
			download = true
		}

		count, err := countAvailableNodesWithModelID(ctx, db, modelID)
		if err != nil {
			return err
		}
		if count < 3 {
			downloadNodes, err := selectNodesForDownloadTask(ctx, task, modelID, 10-int(count))
			if err != nil {
				return err
			}
			if len(downloadNodes) > 0 {
				for _, downloadNode := range downloadNodes {
					if !download || node.Address != downloadNode.Address {
						emitEvent(ctx, db, &models.DownloadModelEvent{
							NodeAddress: downloadNode.Address,
							ModelID:     modelID,
							TaskType:    task.TaskType,
						})
					}
				}
			}
		}
	}
	return nil
}

func checkTaskSelectedNode(ctx context.Context, db *gorm.DB, task *models.InferenceTask) (*models.Node, error) {
	node, err := models.GetNodeByAddress(ctx, db, task.SelectedNode)
	if err != nil {
		return nil, err
	}
	if !(node.CurrentTaskIDCommitment.Valid && node.CurrentTaskIDCommitment.String == task.TaskIDCommitment) {
		return nil, errors.New("node current task is wrong")
	}
	return node, nil
}

func SetTaskStatusScoreReady(ctx context.Context, db *gorm.DB, task *models.InferenceTask) error {
	_, err := checkTaskSelectedNode(ctx, db, task)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		err = task.Update(ctx, tx, &models.InferenceTask{
			Status:         models.TaskScoreReady,
			Score:          task.Score,
			ScoreReadyTime: sql.NullTime{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return err
		}
		return emitEvent(ctx, tx, &models.TaskScoreReadyEvent{
			TaskIDCommitment: task.TaskIDCommitment,
			SelectedNode:     task.SelectedNode,
			Score:            task.Score,
		})
	})
}

func SetTaskStatusErrorReported(ctx context.Context, db *gorm.DB, task *models.InferenceTask) error {
	_, err := checkTaskSelectedNode(ctx, db, task)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		err = task.Update(ctx, tx, &models.InferenceTask{
			Status:         models.TaskErrorReported,
			TaskError:      task.TaskError,
			ScoreReadyTime: sql.NullTime{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return err
		}
		return emitEvent(ctx, tx, &models.TaskErrorReportedEvent{
			TaskIDCommitment: task.TaskIDCommitment,
			SelectedNode:     task.SelectedNode,
			TaskError:        task.TaskError,
		})
	})
}

func SetTaskStatusValidated(ctx context.Context, db *gorm.DB, task *models.InferenceTask) error {
	_, err := checkTaskSelectedNode(ctx, db, task)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		err = task.Update(ctx, tx, &models.InferenceTask{
			Status:        models.TaskValidated,
			ValidatedTime: sql.NullTime{Time: time.Now(), Valid: true},
			QOSScore:      task.QOSScore,
			TaskID:        task.TaskID,
		})
		if err != nil {
			return err
		}
		return emitEvent(ctx, tx, &models.TaskValidatedEvent{TaskIDCommitment: task.TaskIDCommitment, SelectedNode: task.SelectedNode})
	})
}

func SetTaskStatusGroupValidated(ctx context.Context, db *gorm.DB, task *models.InferenceTask) error {
	_, err := checkTaskSelectedNode(ctx, db, task)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		err = task.Update(ctx, tx, &models.InferenceTask{
			Status:        models.TaskGroupValidated,
			ValidatedTime: sql.NullTime{Time: time.Now(), Valid: true},
			QOSScore:      task.QOSScore,
			TaskID:        task.TaskID,
		})
		if err != nil {
			return err
		}
		return emitEvent(ctx, tx, &models.TaskValidatedEvent{TaskIDCommitment: task.TaskIDCommitment, SelectedNode: task.SelectedNode})
	})
}

func SetTaskStatusEndInvalidated(ctx context.Context, db *gorm.DB, task *models.InferenceTask) error {
	node, err := checkTaskSelectedNode(ctx, db, task)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		err = task.Update(ctx, tx, &models.InferenceTask{
			Status:        models.TaskEndInvalidated,
			ValidatedTime: sql.NullTime{Time: time.Now(), Valid: true},
			QOSScore:      task.QOSScore,
			TaskID:        task.TaskID,
		})
		if err != nil {
			return err
		}
		nodeSlash(ctx, tx, node)
		return emitEvent(ctx, tx, &models.TaskEndInvalidatedEvent{TaskIDCommitment: task.TaskIDCommitment, SelectedNode: task.SelectedNode})
	})
}

func SetTaskStatusEndGroupRefund(ctx context.Context, db *gorm.DB, task *models.InferenceTask) error {
	node, err := checkTaskSelectedNode(ctx, db, task)
	if err != nil {
		return err
	}

	appConfig := config.GetConfig()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := Transfer(ctx, tx, appConfig.Blockchain.Account.Address, task.Creator, &task.TaskFee.Int); err != nil {
			return err
		}

		if err := nodeFinishTask(ctx, tx, node); err != nil {
			return err
		}

		err = task.Update(ctx, tx, &models.InferenceTask{
			Status:        models.TaskEndGroupRefund,
			ValidatedTime: sql.NullTime{Time: time.Now(), Valid: true},
			QOSScore:      task.QOSScore,
			TaskID:        task.TaskID,
		})
		if err != nil {
			return err
		}
		return emitEvent(ctx, tx, &models.TaskEndInvalidatedEvent{TaskIDCommitment: task.TaskIDCommitment, SelectedNode: task.SelectedNode})
	})
}

func SetTaskStatusEndAborted(ctx context.Context, db *gorm.DB, task *models.InferenceTask, aboutIssuer string) error {
	if task.Status == models.TaskEndAborted {
		return nil
	}
	if task.Status == models.TaskEndSuccess || task.Status == models.TaskEndInvalidated || task.Status == models.TaskEndGroupSuccess || task.Status == models.TaskEndGroupRefund {
		return errors.New("illegal previous task state")
	}
	lastStatus := task.Status

	newTask := &models.InferenceTask{
		Status:        models.TaskEndAborted,
		AbortReason:   task.AbortReason,
		TaskID:        task.TaskID,
		ValidatedTime: task.ValidatedTime,
	}
	appConfig := config.GetConfig()
	if task.Status != models.TaskQueued {
		node, err := checkTaskSelectedNode(ctx, db, task)
		if err != nil {
			return err
		}

		if lastStatus == models.TaskScoreReady || lastStatus == models.TaskErrorReported {
			newTask.QOSScore = task.QOSScore
		}

		return db.Transaction(func(tx *gorm.DB) error {
			if err := Transfer(ctx, tx, appConfig.Blockchain.Account.Address, task.Creator, &task.TaskFee.Int); err != nil {
				return err
			}

			if err := nodeFinishTask(ctx, tx, node); err != nil {
				return err
			}
			if err := task.Update(ctx, tx, newTask); err != nil {
				return err
			}
			return emitEvent(ctx, tx, &models.TaskEndAbortedEvent{
				TaskIDCommitment: task.TaskIDCommitment,
				AbortIssuer:      aboutIssuer,
				AbortReason:      task.AbortReason,
				LastStatus:       lastStatus,
			})
		})
	} else {
		return db.Transaction(func(tx *gorm.DB) error {
			if err := Transfer(ctx, tx, appConfig.Blockchain.Account.Address, task.Creator, &task.TaskFee.Int); err != nil {
				return err
			}

			if err := task.Update(ctx, tx, newTask); err != nil {
				return err
			}
			return emitEvent(ctx, tx, &models.TaskEndAbortedEvent{
				TaskIDCommitment: task.TaskIDCommitment,
				AbortIssuer:      aboutIssuer,
				AbortReason:      task.AbortReason,
				LastStatus:       lastStatus,
			})
		})

	}
}

func SetTaskStatusEndSuccess(ctx context.Context, db *gorm.DB, task *models.InferenceTask) error {
	node, err := checkTaskSelectedNode(ctx, db, task)
	if err != nil {
		return err
	}

	tasks, err := models.GetTaskGroupByTaskID(ctx, db, task.TaskID)
	if err != nil {
		return err
	}
	status := models.TaskEndSuccess
	if len(tasks) > 1 {
		status = models.TaskEndGroupSuccess
	}
	// calculate each task's payment
	var totalScore uint64 = 0
	var validTasks []models.InferenceTask
	for _, t := range tasks {
		if t.Status == models.TaskValidated || t.Status == models.TaskGroupValidated || t.Status == models.TaskEndGroupRefund {
			totalScore += t.QOSScore
			validTasks = append(validTasks, t)
		}
	}

	payments := map[string]*big.Int{}
	if status == models.TaskEndSuccess {
		payments[task.SelectedNode] = &task.TaskFee.Int
	} else {
		totalRem := big.NewInt(0)
		for i, t := range validTasks {
			payment := big.NewInt(0).Mul(&t.TaskFee.Int, big.NewInt(0).SetUint64(t.QOSScore))
			payment, rem := big.NewInt(0).QuoRem(payment, big.NewInt(0).SetUint64(totalScore), big.NewInt(0))
			totalRem.Add(totalRem, rem)
			if i == len(validTasks)-1 {
				payment.Add(payment, totalRem)
			}
			payments[t.SelectedNode] = payment
		}
	}

	appConfig := config.GetConfig()
	return db.Transaction(func(tx *gorm.DB) error {
		for address, payment := range payments {
			if err := Transfer(ctx, tx, appConfig.Blockchain.Account.Address, address, payment); err != nil {
				return err
			}
		}

		if err := nodeFinishTask(ctx, tx, node); err != nil {
			return err
		}

		err = task.Update(ctx, tx, &models.InferenceTask{
			Status:             status,
			ResultUploadedTime: sql.NullTime{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return err
		}
		if status == models.TaskEndSuccess {
			return emitEvent(ctx, tx, &models.TaskEndSuccessEvent{TaskIDCommitment: task.TaskIDCommitment, SelectedNode: task.SelectedNode})
		} else {
			return emitEvent(ctx, tx, &models.TaskEndGroupSuccessEvent{TaskIDCommitment: task.TaskIDCommitment, SelectedNode: task.SelectedNode})
		}
	})
}
