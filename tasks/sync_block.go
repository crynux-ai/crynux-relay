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
		log.Errorln("SyncedBlocks: error getting synced block from the database")
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
		log.Errorln("SyncedBlocks: error getting synced block from the database")
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

type TxHashWithBlock struct {
	TxHash common.Hash
	Block  *types.Block
}

type TxReceiptWithBlock struct {
	TxReceipt *types.Receipt
	Block     *types.Block
}

func processBlocknum(client *ethclient.Client, blocknumCh <-chan uint64, txHashCh chan<- TxHashWithBlock) {
	for {
		blocknum, ok := <-blocknumCh
		if !ok {
			break
		}
		for {
			log.Debugf("SyncedBlocks: getting block %d", blocknum)
			block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(blocknum)))
			if err != nil {
				log.Errorf("SyncedBlocks: get block %d error: %v", blocknum, err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, tx := range block.Transactions() {
				txHashCh <- TxHashWithBlock{
					TxHash: tx.Hash(),
					Block:  block,
				}
			}
			break
		}
	}
}

func processTxHash(client *ethclient.Client, txHashCh <-chan TxHashWithBlock, txReceiptCh chan<- TxReceiptWithBlock) {
	for {
		txHashWithBlock, ok := <-txHashCh
		txHash := txHashWithBlock.TxHash
		if !ok {
			break
		}

		for {
			log.Debugf("SyncedBlocks: getting tx receipt %s", txHash.Hex())
			receipt, err := client.TransactionReceipt(context.Background(), txHash)
			if err != nil {
				log.Errorf("SyncedBlocks: get transaction receipt of tx %s err: %v", txHash.Hex(), err)
				time.Sleep(time.Second)
				continue
			}
			txReceiptCh <- TxReceiptWithBlock{
				TxReceipt: receipt,
				Block:     txHashWithBlock.Block,
			}
			break
		}
	}
}

func processTxReceipt(txReceiptCh <-chan TxReceiptWithBlock) {
	for {
		receiptWithBlock, ok := <-txReceiptCh
		receipt := receiptWithBlock.TxReceipt
		block := receiptWithBlock.Block
		if !ok {
			break
		}

		for {
			log.Debugf("SyncedBlocks: processing task pending of %s", receipt.TxHash.Hex())
			if err := processTaskPending(receipt); err != nil {
				log.Errorf("SyncedBlocks: processing task pending error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for {
			log.Debugf("SyncedBlocks: processing task started of %s", receipt.TxHash.Hex())
			if err := processTaskStarted(receipt); err != nil {
				log.Errorf("SyncedBlocks: processing task started error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for {
			log.Debugf("SyncedBlocks: processing task success of %s", receipt.TxHash.Hex())
			if err := processTaskSuccess(receipt); err != nil {
				log.Errorf("SyncedBlocks: processing task success error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for {
			log.Debugf("SyncedBlocks: processing task aborted of %s", receipt.TxHash.Hex())
			if err := processTaskAborted(receipt); err != nil {
				log.Errorf("SyncedBlocks: processing task aborted error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for {
			log.Debugf("SyncedBlocks: processing task node success of %s", receipt.TxHash.Hex())
			if err := processTaskNodeSuccess(block, receipt); err != nil {
				log.Errorf("SyncedBlocks: processing task node success error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for {
			log.Debugf("SyncedBlocks: processing task node cancelled of %s", receipt.TxHash.Hex())
			if err := processTaskNodeCancelled(block, receipt); err != nil {
				log.Errorf("SyncedBlocks: processing task node cancelled error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

		for {
			log.Debugf("SyncedBlocks: processing task node slashed of %s", receipt.TxHash.Hex())
			if err := processTaskNodeSlashed(block, receipt); err != nil {
				log.Errorf("SyncedBlocks: processing task node slashed error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}

	}
}

func syncBlocks(client *ethclient.Client, startBlock, endBlock uint64, concurrency int) {
	blocknumCh := make(chan uint64, 10)
	txHashCh := make(chan TxHashWithBlock, 10)
	txReceiptCh := make(chan TxReceiptWithBlock, 10)

	var blocknumWG sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		blocknumWG.Add(1)
		go func() {
			processBlocknum(client, blocknumCh, txHashCh)
			blocknumWG.Done()
		}()
	}

	go func() {
		blocknumWG.Wait()
		close(txHashCh)
		log.Debug("SyncedBlocks: all blocknums have been processed")
	}()

	var txHashWG sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		txHashWG.Add(1)
		go func() {
			processTxHash(client, txHashCh, txReceiptCh)
			txHashWG.Done()
		}()
	}

	go func() {
		txHashWG.Wait()
		close(txReceiptCh)
		log.Debug("SyncedBlocks: all tx hashes have been processed")
	}()

	finishCh := make(chan struct{})
	go func() {
		processTxReceipt(txReceiptCh)
		close(finishCh)
	}()

	for i := startBlock; i < endBlock; i++ {
		blocknumCh <- i
	}
	close(blocknumCh)

	<-finishCh

	log.Debug("SyncedBlocks: all tx receipts have been processed")
}

func processChannel(syncedBlock *models.SyncedBlock) {

	interval := 1

	client, err := blockchain.GetRpcClient()
	if err != nil {
		log.Errorln("SyncedBlocks: error getting the eth rpc client")
		log.Errorln(err)
		time.Sleep(time.Duration(interval) * time.Second)
		return
	}

	latestBlockNum, err := client.BlockNumber(context.Background())
	if err != nil {
		log.Errorln("SyncedBlocks: error getting the latest block number")
		log.Errorln(err)
		time.Sleep(time.Duration(interval) * time.Second)
		return
	}

	if latestBlockNum <= syncedBlock.BlockNumber {
		time.Sleep(time.Duration(interval) * time.Second)
		return
	}

	log.Debugln("SyncedBlocks: new block received: " + strconv.FormatUint(latestBlockNum, 10))

	step := 100

	for start := syncedBlock.BlockNumber + 1; start <= latestBlockNum; start += uint64(step) {
		end := start + uint64(step)
		if end > latestBlockNum+1 {
			end = latestBlockNum + 1
		}

		syncBlocks(client, start, end, 4)

		syncedBlock.BlockNumber = end - 1
		for {
			if err := config.GetDB().Save(syncedBlock).Error; err != nil {
				log.Errorf("SyncedBlocks: save synced block error: %v", err)
				time.Sleep(time.Second)
			}
			log.Debugf("SyncedBlocks: update synced block %d", syncedBlock.BlockNumber)
			break
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

		log.Debugln("SyncedBlocks: Task pending on chain: " +
			taskPending.TaskId.String() +
			"|" + taskPending.Creator.Hex() +
			"|" + string(taskPending.TaskHash[:]) +
			"|" + string(taskPending.DataHash[:]))

		task := &models.InferenceTask{}

		query := &models.InferenceTask{
			TaskId: taskPending.TaskId.Uint64(),
		}

		attributes := &models.InferenceTask{
			Creator:  taskPending.Creator.Hex(),
			TaskHash: hexutil.Encode(taskPending.TaskHash[:]),
			DataHash: hexutil.Encode(taskPending.DataHash[:]),
			Status:   models.InferenceTaskCreatedOnChain,
			TaskType: models.ChainTaskType(taskPending.TaskType.Int64()),
		}

		err = config.GetDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Where(query).Attrs(attributes).FirstOrCreate(task).Error; err != nil {
				return err
			}
			taskStatusLog := models.InferenceTaskStatusLog{
				InferenceTask: *task,
				Status:        models.InferenceTaskCreatedOnChain,
			}
			if err := tx.Create(&taskStatusLog).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
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

		log.Debugln("SyncedBlocks: Task created on chain: " +
			taskStarted.TaskId.String() +
			"|" + taskStarted.Creator.Hex() +
			"|" + string(taskStarted.TaskHash[:]) +
			"|" + string(taskStarted.DataHash[:]))

		taskId := taskStarted.TaskId.Uint64()
		taskStartedEvents[taskId] = append(taskStartedEvents[taskId], models.SelectedNode{NodeAddress: taskStarted.SelectedNode.Hex(), Status: models.NodeStatusPending})
	}

	for taskId, selectedNodes := range taskStartedEvents {
		task := &models.InferenceTask{TaskId: taskId}

		if err := config.GetDB().Where(task).First(task).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}

		taskOnChain, err := blockchain.GetTaskById(taskId)
		if err != nil {
			return err
		}

		vramLimit := taskOnChain.VramLimit.Uint64()
		taskFee, _ := weiToEther(taskOnChain.TotalBalance).Float64()

		if err := config.GetDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(task).Updates(models.InferenceTask{
				VramLimit: vramLimit,
				TaskFee: taskFee,
				Status: models.InferenceTaskStarted,
			}).Error; err != nil {
				return err
			}

			taskStatusLog := models.InferenceTaskStatusLog{
				InferenceTask: *task,
				Status: models.InferenceTaskStarted,
			}
			if err := tx.Create(&taskStatusLog).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}

		var existSelectedNodes []models.SelectedNode
		if err := config.GetDB().Model(task).Association("SelectedNodes").Find(&existSelectedNodes); err != nil {
			return err
		}
		if len(existSelectedNodes) == 0 {
			err := config.GetDB().Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(task).Association("SelectedNodes").Append(selectedNodes); err != nil {
					return err
				}

				var nodeStatusLogs []models.SelectedNodeStatusLog
				for _, selectedNode := range selectedNodes {
					nodeStatusLog := models.SelectedNodeStatusLog{
						SelectedNode: selectedNode,
						Status:       models.NodeStatusPending,
					}
					nodeStatusLogs = append(nodeStatusLogs, nodeStatusLog)
				}

				if err := tx.Create(&nodeStatusLogs).Error; err != nil {
					return err
				}
				return nil
			})
			if err != nil {
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

			err := config.GetDB().Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(task).Association("SelectedNodes").Append(newSelectedNodes); err != nil {
					return err
				}

				var nodeStatusLogs []models.SelectedNodeStatusLog
				for _, selectedNode := range newSelectedNodes {
					nodeStatusLog := models.SelectedNodeStatusLog{
						SelectedNode: selectedNode,
						Status:       models.NodeStatusPending,
					}
					nodeStatusLogs = append(nodeStatusLogs, nodeStatusLog)
				}
				if err := tx.Create(&nodeStatusLogs).Error; err != nil {
					return err
				}
				return nil
			})
			if err != nil {
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

		log.Debugln("SyncedBlocks: Task success on chain: " +
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

		if task.Status != models.InferenceTaskParamsUploaded && task.Status != models.InferenceTaskStarted {
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

		err = config.GetDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(task).Error; err != nil {
				return err
			}
			taskStatusLog := models.InferenceTaskStatusLog{
				InferenceTask: *task,
				Status:        models.InferenceTaskPendingResults,
			}
			if err := tx.Create(&taskStatusLog).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
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

		log.Debugln("SyncedBlocks: Task aborted on chain: " + taskAborted.TaskId.String())

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
		task.AbortReason = taskAborted.Reason

		err = config.GetDB().Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(task).Error; err != nil {
				return err
			}
			taskStatusLog := models.InferenceTaskStatusLog{
				InferenceTask: *task,
				Status:        models.InferenceTaskAborted,
			}
			if err := tx.Create(&taskStatusLog).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func processTaskNodeSuccess(block *types.Block, receipt *types.Receipt) error {
	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	for _, rLog := range receipt.Logs {
		taskNodeSuccess, err := taskContractInstance.ParseTaskNodeSuccess(*rLog)
		if err != nil {
			continue
		}

		taskID := taskNodeSuccess.TaskId.Uint64()
		nodeAddress := taskNodeSuccess.NodeAddress.Hex()
		fee, _ := weiToEther(taskNodeSuccess.Fee).Float64()
		log.Debugf("SyncedBlocks: Node %s succeeded in task %d", nodeAddress, taskID)

		t := time.Unix(int64(block.Time()), 0).Truncate(24 * time.Hour)
		nodeIncentive := models.NodeIncentive{Time: t, NodeAddress: nodeAddress}
		if err := config.GetDB().Model(&nodeIncentive).Where(&nodeIncentive).First(&nodeIncentive).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
		}
		if nodeIncentive.ID > 0 {
			nodeIncentive.Incentive += fee
			nodeIncentive.TaskCount += 1
			if err := config.GetDB().Save(&nodeIncentive).Error; err != nil {
				return err
			}
		} else {
			nodeIncentive.Incentive = fee
			nodeIncentive.TaskCount = 1
			if err := config.GetDB().Create(&nodeIncentive).Error; err != nil {
				return err
			}
		}

		task := models.InferenceTask{TaskId: taskID}
		if err := config.GetDB().Model(&task).Where(&task).Find(&task).Error; err != nil {
			return err
		}
		selectedNode := models.SelectedNode{InferenceTaskID: task.ID, NodeAddress: nodeAddress}
		if err := config.GetDB().Model(&selectedNode).Where(&selectedNode).Find(&selectedNode).Error; err != nil {
			return err
		}
		if err := config.GetDB().Transaction(func(tx *gorm.DB) error {
			selectedNode.Status = models.NodeStatusSuccess
			if err := tx.Save(&selectedNode).Error; err != nil {
				return err
			}
			nodeStatusLog := models.SelectedNodeStatusLog{
				SelectedNode: selectedNode,
				Status:       models.NodeStatusSuccess,
			}
			if err := tx.Create(&nodeStatusLog).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func processTaskNodeCancelled(_ *types.Block, receipt *types.Receipt) error {
	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	for _, rLog := range receipt.Logs {
		taskNodeSuccess, err := taskContractInstance.ParseTaskNodeCancelled(*rLog)
		if err != nil {
			continue
		}

		taskID := taskNodeSuccess.TaskId.Uint64()
		nodeAddress := taskNodeSuccess.NodeAddress.Hex()
		log.Debugf("SyncedBlocks: Node %s cancelled in task %d", nodeAddress, taskID)

		task := models.InferenceTask{TaskId: taskID}
		if err := config.GetDB().Model(&task).Where(&task).Find(&task).Error; err != nil {
			return err
		}
		selectedNode := models.SelectedNode{InferenceTaskID: task.ID, NodeAddress: nodeAddress}
		if err := config.GetDB().Model(&selectedNode).Where(&selectedNode).Find(&selectedNode).Error; err != nil {
			return err
		}
		if err := config.GetDB().Transaction(func(tx *gorm.DB) error {
			selectedNode.Status = models.NodeStatusCancelled
			if err := tx.Save(&selectedNode).Error; err != nil {
				return err
			}
			nodeStatusLog := models.SelectedNodeStatusLog{
				SelectedNode: selectedNode,
				Status:       models.NodeStatusCancelled,
			}
			if err := tx.Create(&nodeStatusLog).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func processTaskNodeSlashed(_ *types.Block, receipt *types.Receipt) error {
	taskContractInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return err
	}

	for _, rLog := range receipt.Logs {
		taskNodeSuccess, err := taskContractInstance.ParseTaskNodeSlashed(*rLog)
		if err != nil {
			continue
		}

		taskID := taskNodeSuccess.TaskId.Uint64()
		nodeAddress := taskNodeSuccess.NodeAddress.Hex()
		log.Debugf("SyncedBlocks: Node %s slashed in task %d", nodeAddress, taskID)

		task := models.InferenceTask{TaskId: taskID}
		if err := config.GetDB().Model(&task).Where(&task).Find(&task).Error; err != nil {
			return err
		}
		selectedNode := models.SelectedNode{InferenceTaskID: task.ID, NodeAddress: nodeAddress}
		if err := config.GetDB().Model(&selectedNode).Where(&selectedNode).Find(&selectedNode).Error; err != nil {
			return err
		}
		if err := config.GetDB().Transaction(func(tx *gorm.DB) error {
			selectedNode.Status = models.NodeStatusSlashed
			if err := tx.Save(&selectedNode).Error; err != nil {
				return err
			}
			nodeStatusLog := models.SelectedNodeStatusLog{
				SelectedNode: selectedNode,
				Status:       models.NodeStatusSlashed,
			}
			if err := tx.Create(&nodeStatusLog).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
