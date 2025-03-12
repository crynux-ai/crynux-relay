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
	task.TaskID = taskID

	if err := validateVRFProof(task.SamplingSeed, vrfProof, publicKey, task.Creator, false); err != nil {
		return err
	}

	if task.Status == models.TaskScoreReady {
		task.QOSScore = getTaskQosScore(0)
		return SetTaskStatusValidated(ctx, config.GetDB(), task)
	} else {
		task.AbortReason = models.TaskAbortIncorrectResult
		task.ValidatedTime = sql.NullTime{Time: time.Now(), Valid: true}
		return SetTaskStatusEndAborted(ctx, config.GetDB(), task, task.Creator)
	}
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
			distance := utils.HammingDistance(h1, h2)
			return uint64(distance) < threshold
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
		task.TaskID = taskID
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
		score := getTaskQosScore(i)
		task.QOSScore = score
	}

	// validate tasks' score
	appConfig := config.GetConfig()

	isValid := []bool{false, false, false}
	for i := 0; i < 2; i++ {
		for j := i + 1; j < 3; j++ {
			if tasks[i].Status != models.TaskEndAborted && tasks[j].Status != models.TaskEndAborted {
				if compareTaskScore(tasks[i], tasks[j], appConfig.Task.DistanceThreshold) {
					isValid[i] = true
					isValid[j] = true
				}
			}
		}
	}

	return config.GetDB().Transaction(func(tx *gorm.DB) error {
		emitValidated := false

		for i, task := range tasks {
			if task.Status == models.TaskEndAborted {
				task.AbortReason = models.TaskAbortIncorrectResult
				task.ValidatedTime = sql.NullTime{Time: time.Now(), Valid: true}
				if err := SetTaskStatusEndAborted(ctx, tx, task, task.Creator); err != nil {
					return err
				}
			}
			if isValid[i] {
				if !emitValidated {
					if err := SetTaskStatusGroupValidated(ctx, tx, task); err != nil {
						return err
					}
					emitValidated = true
				} else {
					if err := SetTaskStatusEndGroupRefund(ctx, tx, task); err != nil {
						return err
					}
				}
			} else {
				if err := SetTaskStatusEndInvalidated(ctx, tx, task); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
