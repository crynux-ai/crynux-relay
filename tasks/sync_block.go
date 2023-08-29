package tasks

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/blockchain"
	"h_relay/config"
	"h_relay/models"
	"time"
)

func StartSyncBlockWithTerminateChannel(ch <-chan int) {
	appConfig := config.GetConfig()
	syncedBlock := &models.SyncedBlock{}

	if err := config.GetDB().First(&syncedBlock).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			syncedBlock.BlockNumber = appConfig.Blockchain.StartBlockNum
		} else {
			log.Fatal(err)
		}
	}

	client, err := blockchain.GetWebSocketClient()
	if err != nil {
		log.Fatal(err)
	}

	headers := make(chan *types.Header)

	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case stop := <-ch:
			if stop == 1 {
				return
			} else {
				processChannel(sub, headers, syncedBlock)
			}
		default:
			processChannel(sub, headers, syncedBlock)
		}
	}
}

func StartSyncBlock() {
	appConfig := config.GetConfig()
	syncedBlock := &models.SyncedBlock{}

	if err := config.GetDB().First(&syncedBlock).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			syncedBlock.BlockNumber = appConfig.Blockchain.StartBlockNum
		} else {
			log.Fatal(err)
		}
	}

	client, err := blockchain.GetWebSocketClient()
	if err != nil {
		log.Fatal(err)
	}

	headers := make(chan *types.Header)

	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}

	for {
		processChannel(sub, headers, syncedBlock)
	}
}

func processChannel(sub ethereum.Subscription, headers chan *types.Header, syncedBlock *models.SyncedBlock) {

	interval := 1

	select {
	case err := <-sub.Err():
		log.Errorln(err)
		time.Sleep(time.Duration(interval) * time.Second)
	case header := <-headers:

		currentBlockNum := header.Number

		log.Debugln("new block received: " + header.Number.String())

		if err := processTaskCreated(syncedBlock.BlockNumber+1, currentBlockNum.Uint64()); err != nil {
			log.Errorln(err)
			time.Sleep(time.Duration(interval) * time.Second)
			return
		}

		if err := processTaskSuccess(syncedBlock.BlockNumber+1, currentBlockNum.Uint64()); err != nil {
			log.Errorln(err)
			time.Sleep(time.Duration(interval) * time.Second)
			return
		}

		oldNum := syncedBlock.BlockNumber
		syncedBlock.BlockNumber = currentBlockNum.Uint64()
		if err := config.GetDB().Save(syncedBlock).Error; err != nil {
			syncedBlock.BlockNumber = oldNum
			log.Errorln(err)
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}

	time.Sleep(time.Duration(interval) * time.Second)
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

		attributes := &models.InferenceTask{
			Creator:  taskCreated.Creator.Hex(),
			TaskHash: hexutil.Encode(taskCreated.TaskHash[:]),
			DataHash: hexutil.Encode(taskCreated.DataHash[:]),
			Status:   models.InferenceTaskCreatedOnChain,
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
	}

	if err := taskSuccessEventIterator.Close(); err != nil {
		return err
	}

	return nil
}
