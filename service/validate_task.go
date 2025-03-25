package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"database/sql"
	"errors"
	"math/big"
	"sort"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vechain/go-ecvrf"
	"gorm.io/gorm"
)

func validateTaskID(taskID, nonce, taskIDCommitment string) error {
	taskIDBytes, err := hexutil.Decode(taskID)
	if err != nil {
		return err
	}
	nonceBytes, err := hexutil.Decode(nonce)
	if err != nil {
		return err
	}

	hash := crypto.Keccak256Hash(append(taskIDBytes, nonceBytes...))
	if hash.Hex() != taskIDCommitment {
		return errors.New("task id incorrect")
	}
	return nil
}

func validateVRFProof(samplingSeed, vrfProof, publicKey string, creator string, grouped bool) error {
	samplingSeedBytes, err := hexutil.Decode(samplingSeed)
	if err != nil {
		return errors.New("invalid sampling seed")
	}

	pkBytes, err := hexutil.Decode(publicKey)
	if err != nil {
		return errors.New("invalid public key")
	}
	if len(pkBytes) != 64 {
		return errors.New("invalid public key")
	}
	vrfProofBytes, err := hexutil.Decode(vrfProof)
	if err != nil {
		return errors.New("invalid vrf proof")
	}
	if len(vrfProofBytes) != 81 {
		return errors.New("invalid vrf proof")
	}

	pkBytes = append([]byte{0x04}, pkBytes...)
	pubKey, err := secp256k1.ParsePubKey(pkBytes)
	if err != nil {
		return err
	}
	ecdsaPubKey := pubKey.ToECDSA()
	address := crypto.PubkeyToAddress(*ecdsaPubKey)
	if address.Hex() != creator {
		return errors.New("not task creator")
	}
	beta, err := ecvrf.Secp256k1Sha256Tai.Verify(pubKey.ToECDSA(), samplingSeedBytes, vrfProofBytes)
	if err != nil {
		return err
	}
	number := big.NewInt(0).SetBytes(beta)
	r := big.NewInt(0).Mod(number, big.NewInt(10)).Uint64()
	if grouped && r != 0 {
		return errors.New("task is not selected for validation")
	}
	if !grouped && r == 0 {
		return errors.New("task is selected for validation")
	}
	return nil
}

func ValidateSingleTask(ctx context.Context, task *models.InferenceTask, taskID, vrfProof, publicKey string) error {
	if !(task.Status == models.TaskScoreReady || task.Status == models.TaskErrorReported) {
		return errors.New("illegal task state")
	}

	if err := validateTaskID(taskID, task.Nonce, task.TaskIDCommitment); err != nil {
		return err
	}
	if err := task.Update(ctx, config.GetDB(), map[string]interface{}{"task_id": taskID}); err != nil {
		return err
	}

	if err := validateVRFProof(task.SamplingSeed, vrfProof, publicKey, task.Creator, false); err != nil {
		return err
	}

	if task.Status == models.TaskScoreReady {
		return SetTaskStatusValidated(ctx, config.GetDB(), task)
	} else {
		task.AbortReason = models.TaskAbortIncorrectResult
		task.ValidatedTime = sql.NullTime{Time: time.Now(), Valid: true}
		return SetTaskStatusEndAborted(ctx, config.GetDB(), task, task.Creator)
	}
}

func checkHammingDistance(h1, h2 []byte, threshold uint64) bool {
	if len(h1) != len(h2) || len(h1)%8 != 0 {
		return false
	}

	for i := 0; i < len(h1); i += 8 {
		distance := utils.HammingDistance(h1[i:i+8], h2[i:i+8])
		if uint64(distance) >= threshold {
			return false
		}
	}

	return true
}

func compareTaskScore(task1, task2 *models.InferenceTask, threshold uint64) bool {
	if task1.TaskType != task2.TaskType {
		return false
	}
	if task1.Status != task2.Status {
		return false
	}
	if task1.Status == models.TaskScoreReady {
		if task1.TaskType == models.TaskTypeSD || task1.TaskType == models.TaskTypeSDFTLora {
			h1 := hexutil.MustDecode(task1.Score)
			h2 := hexutil.MustDecode(task2.Score)
			return checkHammingDistance(h1, h2, threshold)
		} else if task1.TaskType == models.TaskTypeLLM {
			return task1.Score == task2.Score
		} else {
			return false
		}
	} else {
		return true
	}
}

func ValidateTaskGroup(ctx context.Context, tasks []*models.InferenceTask, taskID, vrfProof, publicKey string) error {
	if len(tasks) != 3 {
		return errors.New("task group size is not 3")
	}

	for _, task := range tasks {
		if !(task.Status == models.TaskScoreReady || task.Status == models.TaskErrorReported || task.Status == models.TaskEndAborted) {
			return errors.New("illegal task state")
		}
	}

	for _, task := range tasks {
		if err := validateTaskID(taskID, task.Nonce, task.TaskIDCommitment); err != nil {
			return err
		}
	}
	if err := config.GetDB().Transaction(func(tx *gorm.DB) error {
		for _, task := range tasks {
			if err := task.Update(ctx, tx, map[string]interface{}{"task_id": taskID}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// sort tasks by sequence
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})
	// validate vrf proof
	samplingSeed := tasks[0].SamplingSeed
	for _, task := range tasks {
		if err := validateVRFProof(samplingSeed, vrfProof, publicKey, task.Creator, true); err != nil {
			return err
		}
	}

	// sort tasks by time cost
	sort.Slice(tasks, func(i, j int) bool {
		ti := tasks[i].ExecutionTime()
		tj := tasks[j].ExecutionTime()
		if ti == tj {
			return tasks[i].ID < tasks[j].ID
		}
		return ti < tj
	})
	// set task qos score
	for i, task := range tasks {
		if task.Status != models.TaskEndAborted {
			score := getTaskQosScore(i)
			task.QOSScore = score
		}
	}

	// validate tasks' score
	appConfig := config.GetConfig()

	nextStatusMap := make(map[string]models.TaskStatus)
	finishedTasks := make([]*models.InferenceTask, 0)
	for _, task := range tasks {
		nextStatusMap[task.TaskIDCommitment] = models.TaskEndAborted
		if task.Status != models.TaskEndAborted {
			finishedTasks = append(finishedTasks, task)
		}
	}

	if len(finishedTasks) == 2 {
		if compareTaskScore(finishedTasks[0], finishedTasks[1], appConfig.Task.DistanceThreshold) && finishedTasks[0].Status == models.TaskScoreReady {
			nextStatusMap[finishedTasks[0].TaskIDCommitment] = models.TaskGroupValidated
			nextStatusMap[finishedTasks[1].TaskIDCommitment] = models.TaskEndGroupRefund
		}
	} else if len(finishedTasks) == 3 {
		same1 := compareTaskScore(finishedTasks[0], finishedTasks[1], appConfig.Task.DistanceThreshold)
		same2 := compareTaskScore(finishedTasks[0], finishedTasks[2], appConfig.Task.DistanceThreshold)
		same3 := compareTaskScore(finishedTasks[1], finishedTasks[2], appConfig.Task.DistanceThreshold)
		if same1 {
			if finishedTasks[0].Status == models.TaskScoreReady {
				nextStatusMap[finishedTasks[0].TaskIDCommitment] = models.TaskGroupValidated
				nextStatusMap[finishedTasks[1].TaskIDCommitment] = models.TaskEndGroupRefund
			}
			if same2 {
				nextStatusMap[finishedTasks[2].TaskIDCommitment] = models.TaskEndGroupRefund
			} else {
				nextStatusMap[finishedTasks[2].TaskIDCommitment] = models.TaskEndInvalidated
			}
		} else if same2 {
			if finishedTasks[0].Status == models.TaskScoreReady {
				nextStatusMap[finishedTasks[0].TaskIDCommitment] = models.TaskGroupValidated
				nextStatusMap[finishedTasks[2].TaskIDCommitment] = models.TaskEndGroupRefund
			}
			nextStatusMap[finishedTasks[1].TaskIDCommitment] = models.TaskEndInvalidated
		} else if same3 {
			if finishedTasks[1].Status == models.TaskScoreReady {
				nextStatusMap[finishedTasks[1].TaskIDCommitment] = models.TaskGroupValidated
				nextStatusMap[finishedTasks[2].TaskIDCommitment] = models.TaskEndGroupRefund
			}
			nextStatusMap[finishedTasks[0].TaskIDCommitment] = models.TaskEndInvalidated
		}
	}

	return config.GetDB().Transaction(func(tx *gorm.DB) error {
		for _, task := range tasks {
			nextStatus := nextStatusMap[task.TaskIDCommitment]

			if nextStatus == models.TaskEndInvalidated {
				if err := SetTaskStatusEndInvalidated(ctx, tx, task); err != nil {
					return err
				}
			} else if nextStatus == models.TaskGroupValidated {
				if err := SetTaskStatusGroupValidated(ctx, tx, task); err != nil {
					return err
				}
			} else if nextStatus == models.TaskEndGroupRefund {
				if err := SetTaskStatusEndGroupRefund(ctx, tx, task); err != nil {
					return err
				}
			} else {
				task.AbortReason = models.TaskAbortIncorrectResult
				task.ValidatedTime = sql.NullTime{Time: time.Now(), Valid: true}
				if err := SetTaskStatusEndAborted(ctx, tx, task, task.Creator); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
