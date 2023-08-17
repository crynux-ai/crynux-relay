package tasks

import (
	"context"
	"errors"
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

func StartSyncBlock() {
	interval := 1
	appConfig := config.GetConfig()
	syncedBlock := &models.SyncedBlock{}

	if err := config.GetDB().First(&syncedBlock).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
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
		case err := <-sub.Err():
			log.Errorln(err)
			time.Sleep(time.Duration(interval) * time.Second)
		case header := <-headers:

			currentBlockNum := header.Number

			if err := processTaskCreated(syncedBlock.BlockNumber+1, currentBlockNum.Uint64()); err != nil {
				log.Errorln(err)
				time.Sleep(time.Duration(interval) * time.Second)
				continue
			}

			oldNum := syncedBlock.BlockNumber
			syncedBlock.BlockNumber = header.Number.Uint64()
			if err := config.GetDB().Save(&syncedBlock).Error; err != nil {
				syncedBlock.BlockNumber = oldNum
				log.Errorln(err)
				time.Sleep(time.Duration(interval) * time.Second)
			}
		}
	}
}

func processTaskCreated(startBlockNum, endBlockNum uint64) error {

	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), time.Duration(3)*time.Second)
	defer cancelFn()

	taskCreatedEventIterator, err := taskContractInstance.FilterTaskCreated(
		&bind.FilterOpts{
			Start:   startBlockNum,
			End:     &endBlockNum,
			Context: ctx,
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

		task := &models.InferenceTask{
			TaskId:   taskCreated.TaskId.Uint64(),
			Creator:  taskCreated.Creator.Hex(),
			TaskHash: hexutil.Encode(taskCreated.TaskHash[:]),
			DataHash: hexutil.Encode(taskCreated.DataHash[:]),
			Status:   models.InferenceTaskCreatedOnChain,
		}

		if err := config.GetDB().Create(&task).Error; err != nil {
			if !errors.Is(err, gorm.ErrDuplicatedKey) {
				return err
			}
		}

		association := config.GetDB().Model(&task).Association("SelectedNodes")

		if err := association.Append(&models.SelectedNode{NodeAddress: taskCreated.SelectedNode.Hex()}); err != nil {
			return err
		}
	}

	if err := taskCreatedEventIterator.Close(); err != nil {
		return err
	}

	return nil
}
