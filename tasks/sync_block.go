package tasks

import (
	"context"
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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

func processBlocknum(client *ethclient.Client, blocknumCh <-chan uint64, txHashCh chan<- common.Hash) {
	for {
		blocknum, ok := <-blocknumCh
		if !ok {
			break
		}
		for {
			log.Debugf("getting block %d", blocknum)
			block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(blocknum)))
			if err != nil {
				log.Errorf("get block %d error: %v", blocknum, err)
				time.Sleep(1 * time.Second)
				continue
			}
	
			for _, tx := range block.Transactions() {
				txHashCh <- tx.Hash()
			}
			break
		}
	}
}

func processTxHash(client *ethclient.Client, txHashCh <-chan common.Hash, txReceiptCh chan<- *types.Receipt) {
	for {
		txHash, ok := <-txHashCh
		if !ok {
			break
		}
		
		for {
			log.Debugf("getting tx receipt %s", txHash.Hex())
			receipt, err := client.TransactionReceipt(context.Background(), txHash)
			if err != nil {
				log.Errorf("get transaction receipt of tx %s err: %v", txHash.Hex(), err)
				time.Sleep(time.Second)
				continue
			}
			txReceiptCh <- receipt
			break
		}
	}
}

func processTxReceipt(txReceiptCh <-chan *types.Receipt) {
	for {
		receipt, ok := <-txReceiptCh
		if !ok {
			break
		}
		
		for {
			log.Debugf("processing task pending of %s", receipt.TxHash.Hex())
			if err := processTaskPending(receipt); err != nil {
				log.Errorf("processing task pending error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for {
			log.Debugf("processing task started of %s", receipt.TxHash.Hex())
			if err := processTaskStarted(receipt); err != nil {
				log.Errorf("processing task started error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for {
			log.Debugf("processing task success of %s", receipt.TxHash.Hex())
			if err := processTaskSuccess(receipt); err != nil {
				log.Errorf("processing task success error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for {
			log.Debugf("processing task aborted of %s", receipt.TxHash.Hex())
			if err := processTaskAborted(receipt); err != nil {
				log.Errorf("processing task aborted error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}
}

func processChannel(syncedBlock *models.SyncedBlock) {

	interval := 1

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

	blocknumCh := make(chan uint64, 10)
	txHashCh := make(chan common.Hash, 10)
	txReceiptCh := make(chan *types.Receipt, 10)

	concurrency := 4
	step := 100

	for start := syncedBlock.BlockNumber + 1; start <= latestBlockNum; start += uint64(step) {
		end := start + uint64(step)
		if end > latestBlockNum + 1 {
			end = latestBlockNum + 1
		}

		var blocknumWG sync.WaitGroup
		for i := 0; i < concurrency; i++ {
			blocknumWG.Add(1)
			go func ()  {
				processBlocknum(client, blocknumCh, txHashCh)
				blocknumWG.Done()
			}()
		}

		go func ()  {
			blocknumWG.Wait()
			close(txHashCh)
		}()

		var txHashWG sync.WaitGroup
		for i := 0; i < concurrency; i++ {
			txHashWG.Add(1)
			go func ()  {
				processTxHash(client, txHashCh, txReceiptCh)
				txHashWG.Done()
			}()
		}

		go func() {
			txHashWG.Wait()
			close(txReceiptCh)
		}()

		for i := start; i < end; i++ {
			blocknumCh <- i
		}
		close(blocknumCh)

		processTxReceipt(txReceiptCh)

		oldNum := syncedBlock.BlockNumber

		syncedBlock.BlockNumber = latestBlockNum
		if err := config.GetDB().Save(syncedBlock).Error; err != nil {
			syncedBlock.BlockNumber = oldNum
			log.Errorln(err)
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}

	time.Sleep(time.Duration(interval) * time.Second)
}

func processTaskPending(receipt *types.Receipt) error {
	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	for _, rLog := range receipt.Logs {
		taskPending, err := taskContractInstance.ParseTaskPending(*rLog)
		if err != nil {
			continue
		}

		log.Debugln("Task pending on chain: " +
			taskPending.TaskId.String() +
			"|" + taskPending.Creator.Hex() +
			"|" + string(taskPending.TaskHash[:]) +
			"|" + string(taskPending.DataHash[:]))

		task := &models.InferenceTask{}

		query := &models.InferenceTask{
			TaskId: taskPending.TaskId.Uint64(),
		}

		taskOnChain, err := blockchain.GetTaskById(taskPending.TaskId.Uint64())
		if err != nil {
			return err
		}

		attributes := &models.InferenceTask{
			Creator:   taskPending.Creator.Hex(),
			TaskHash:  hexutil.Encode(taskPending.TaskHash[:]),
			DataHash:  hexutil.Encode(taskPending.DataHash[:]),
			Status:    models.InferenceTaskCreatedOnChain,
			TaskType:  models.ChainTaskType(taskPending.TaskType.Int64()),
			VramLimit: taskOnChain.VramLimit.Uint64(),
		}

		if err := config.GetDB().Where(query).Attrs(attributes).FirstOrCreate(task).Error; err != nil {
			return err
		}
	}

	return nil
}

func processTaskStarted(receipt *types.Receipt) error {

	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	taskStartedEvents := make(map[uint64][]models.SelectedNode)

	for _, rLog := range receipt.Logs {
		taskStarted, err := taskContractInstance.ParseTaskStarted(*rLog)
		if err != nil {
			continue
		}

		log.Debugln("Task created on chain: " +
			taskStarted.TaskId.String() +
			"|" + taskStarted.Creator.Hex() +
			"|" + string(taskStarted.TaskHash[:]) +
			"|" + string(taskStarted.DataHash[:]))

		taskId := taskStarted.TaskId.Uint64()
		taskStartedEvents[taskId] = append(taskStartedEvents[taskId], models.SelectedNode{NodeAddress: taskStarted.SelectedNode.Hex()})
	}

	for taskId, selectedNodes := range taskStartedEvents {
		task := &models.InferenceTask{TaskId: taskId}

		if err := config.GetDB().Where(task).First(task).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}

		var existSelectedNodes []models.SelectedNode
		if err := config.GetDB().Model(task).Association("SelectedNodes").Find(&existSelectedNodes); err != nil {
			return err
		}
		if len(existSelectedNodes) == 0 {
			if err := config.GetDB().Model(task).Association("SelectedNodes").Append(selectedNodes); err != nil {
				return err
			}
		} else {
			existNodeAddresses := make(map[string]interface{})
			for _, node := range existSelectedNodes {
				existNodeAddresses[node.NodeAddress] = nil
			}

			var newSelectedNodes []models.SelectedNode
			for _, node := range selectedNodes {
				_, ok := existNodeAddresses[node.NodeAddress]
				if !ok {
					newSelectedNodes = append(newSelectedNodes, node)
				}
			}

			if err := config.GetDB().Model(task).Association("SelectedNodes").Append(newSelectedNodes); err != nil {
				return err
			}
		}

	}

	return nil
}

func processTaskSuccess(receipt *types.Receipt) error {
	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	for _, rLog := range receipt.Logs {
		taskSuccess, err := taskContractInstance.ParseTaskSuccess(*rLog)
		if err != nil {
			continue
		}

		log.Debugln("Task success on chain: " +
			taskSuccess.TaskId.String() +
			"|" + string(taskSuccess.Result) +
			"|" + taskSuccess.ResultNode.Hex())

		task := &models.InferenceTask{
			TaskId: taskSuccess.TaskId.Uint64(),
		}

		if err := config.GetDB().Where(task).First(task).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}

		if task.Status != models.InferenceTaskParamsUploaded {
			continue
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

	return nil
}

func processTaskAborted(receipt *types.Receipt) error {
	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	for _, rLog := range receipt.Logs {
		taskAborted, err := taskContractInstance.ParseTaskAborted(*rLog)
		if err != nil {
			continue
		}

		log.Debugln("Task aborted on chain: " + taskAborted.TaskId.String())

		task := &models.InferenceTask{
			TaskId: taskAborted.TaskId.Uint64(),
		}

		if err := config.GetDB().Where(task).First(task).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}

		if task.Status == models.InferenceTaskResultsUploaded {
			continue
		}

		task.Status = models.InferenceTaskAborted

		if err := config.GetDB().Save(task).Error; err != nil {
			return err
		}
	}

	return nil
}
