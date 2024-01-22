package tasks

import (
	"context"
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func StartSyncBlockWithTerminateChannel(ch <-chan int) {

	syncedBlock, err := getSyncedBlock()

	if err != nil {
		log.Errorln("error getting synced block from the database")
		log.Fatal(err)
	}

	for {
		select {
		case stop := <-ch:
			if stop == 1 {
				return
			} else {
				processChannel(syncedBlock)
			}
		default:
			processChannel(syncedBlock)
		}
	}
}

func StartSyncBlock() {

	syncedBlock, err := getSyncedBlock()

	if err != nil {
		log.Errorln("error getting synced block from the database")
		log.Fatal(err)
	}

	for {
		processChannel(syncedBlock)
	}
}

func getSyncedBlock() (*models.SyncedBlock, error) {
	appConfig := config.GetConfig()
	syncedBlock := &models.SyncedBlock{}

	if err := config.GetDB().First(&syncedBlock).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			syncedBlock.BlockNumber = appConfig.Blockchain.StartBlockNum
		} else {
			return nil, err
		}
	}

	return syncedBlock, nil
}

func processChannel(syncedBlock *models.SyncedBlock) {

	interval := 1
	batchSize := uint64(500)

	client, err := blockchain.GetRpcClient()
	if err != nil {
		log.Errorln("error getting the eth rpc client")
		log.Errorln(err)
		time.Sleep(time.Duration(interval) * time.Second)
		return
	}

	latestBlockNum, err := client.BlockNumber(context.Background())
	if err != nil {
		log.Errorln("error getting the latest block number")
		log.Errorln(err)
		time.Sleep(time.Duration(interval) * time.Second)
		return
	}

	if latestBlockNum <= syncedBlock.BlockNumber {
		time.Sleep(time.Duration(interval) * time.Second)
		return
	}

	log.Debugln("new block received: " + strconv.FormatUint(latestBlockNum, 10))

	for start := syncedBlock.BlockNumber + 1; start <= latestBlockNum; start += batchSize {

		end := start + batchSize - 1

		if end > latestBlockNum {
			end = latestBlockNum
		}

		log.Debugln("processing blocks from " +
			strconv.FormatUint(start, 10) +
			" to " +
			strconv.FormatUint(end, 10) +
			" / " +
			strconv.FormatUint(latestBlockNum, 10))

		if err := processTaskCreated(start, end); err != nil {
			log.Errorln(err)
			time.Sleep(time.Duration(interval) * time.Second)
			return
		}

		if err := processTaskSuccess(start, end); err != nil {
			log.Errorln(err)
			time.Sleep(time.Duration(interval) * time.Second)
			return
		}

		if err := processTaskAborted(start, end); err != nil {
			log.Errorln(err)
			time.Sleep(time.Duration(interval) * time.Second)
			return
		}

		oldNum := syncedBlock.BlockNumber
		syncedBlock.BlockNumber = end
		if err := config.GetDB().Save(syncedBlock).Error; err != nil {
			syncedBlock.BlockNumber = oldNum
			log.Errorln(err)
			time.Sleep(time.Duration(interval) * time.Second)
		}

		if end != latestBlockNum {
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}

	time.Sleep(time.Duration(interval) * time.Second * 3)
}

func processTaskCreated(startBlockNum, endBlockNum uint64) error {

	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	taskCreatedEventIterator, err := taskContractInstance.FilterTaskCreated(
		&bind.FilterOpts{
			Start:   startBlockNum,
			End:     &endBlockNum,
			Context: context.Background(),
		},
		nil,
		nil,
	)

	if err != nil {
		return err
	}

	for {
		if !taskCreatedEventIterator.Next() {
			break
		}

		taskCreated := taskCreatedEventIterator.Event

		log.Debugln("Task created on chain: " +
			taskCreated.TaskId.String() +
			"|" + taskCreated.Creator.Hex() +
			"|" + string(taskCreated.TaskHash[:]) +
			"|" + string(taskCreated.DataHash[:]))

		task := &models.InferenceTask{}

		query := &models.InferenceTask{
			TaskId: taskCreated.TaskId.Uint64(),
		}

		taskOnChain, err := blockchain.GetTaskById(taskCreated.TaskId.Uint64())
		if err != nil {
			return err
		}

		attributes := &models.InferenceTask{
			Creator:   taskCreated.Creator.Hex(),
			TaskHash:  hexutil.Encode(taskCreated.TaskHash[:]),
			DataHash:  hexutil.Encode(taskCreated.DataHash[:]),
			Status:    models.InferenceTaskCreatedOnChain,
			TaskType:  models.ChainTaskType(taskCreated.TaskType.Int64()),
			VramLimit: taskOnChain.VramLimit.Uint64(),
		}

		if err := config.GetDB().Where(query).Attrs(attributes).FirstOrCreate(task).Error; err != nil {
			return err
		}

		association := config.GetDB().Model(task).Association("SelectedNodes")

		if err := association.Append(&models.SelectedNode{NodeAddress: taskCreated.SelectedNode.Hex()}); err != nil {
			return err
		}
	}

	if err := taskCreatedEventIterator.Close(); err != nil {
		return err
	}

	return nil
}

func processTaskSuccess(startBlockNum, endBlockNum uint64) error {
	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	taskSuccessEventIterator, err := taskContractInstance.FilterTaskSuccess(
		&bind.FilterOpts{
			Start:   startBlockNum,
			End:     &endBlockNum,
			Context: context.Background(),
		},
		nil,
	)

	if err != nil {
		return err
	}

	for {
		if !taskSuccessEventIterator.Next() {
			break
		}

		taskSuccess := taskSuccessEventIterator.Event

		log.Debugln("Task success on chain: " +
			taskSuccess.TaskId.String() +
			"|" + string(taskSuccess.Result) +
			"|" + taskSuccess.ResultNode.Hex())

		task := &models.InferenceTask{
			TaskId: taskSuccess.TaskId.Uint64(),
		}

		if err := config.GetDB().Where(task).First(task).Error; err != nil {
			return err
		}

		selectedNode := &models.SelectedNode{
			InferenceTaskID: task.ID,
			NodeAddress:     taskSuccess.ResultNode.Hex(),
		}

		if err := config.GetDB().Where(selectedNode).First(selectedNode).Error; err != nil {
			return err
		}

		selectedNode.Result = hexutil.Encode(taskSuccess.Result)
		selectedNode.IsResultSelected = true

		if err := config.GetDB().Model(selectedNode).Select("Result", "IsResultSelected").Updates(selectedNode).Error; err != nil {
			return err
		}

		task.Status = models.InferenceTaskPendingResults

		if err := config.GetDB().Save(task).Error; err != nil {
			return err
		}
	}

	if err := taskSuccessEventIterator.Close(); err != nil {
		return err
	}

	return nil
}

func processTaskAborted(startBlockNum, endBlockNum uint64) error {
	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	taskAbortedEventIterator, err := taskContractInstance.FilterTaskAborted(
		&bind.FilterOpts{
			Start:   startBlockNum,
			End:     &endBlockNum,
			Context: context.Background(),
		},
	)

	if err != nil {
		return err
	}

	for {
		if !taskAbortedEventIterator.Next() {
			break
		}

		taskAborted := taskAbortedEventIterator.Event

		log.Debugln("Task aborted on chain: " + taskAborted.TaskId.String())

		task := &models.InferenceTask{
			TaskId: taskAborted.TaskId.Uint64(),
		}

		if err := config.GetDB().Where(task).First(task).Error; err != nil {
			return err
		}

		task.Status = models.InferenceTaskAborted

		if err := config.GetDB().Save(task).Error; err != nil {
			return err
		}
	}

	if err := taskAbortedEventIterator.Close(); err != nil {
		return err
	}

	return nil
}
