package tasks

import (
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	log "github.com/sirupsen/logrus"
)

func StartSyncNetworkWithTerminalChannel(ch <-chan int) {
	for {
		select {
		case stop := <-ch:
			if stop == 1 {
				return
			} else {
				SyncNetwork()
			}
		default:
			SyncNetwork()
		}
		time.Sleep(60 * time.Second)
	}
}

func StartSyncNetwork() {
	for {
		SyncNetwork()
		time.Sleep(60 * time.Second)
	}
}

func SyncNetwork() error {
	busyNodes, allNodes, activeNodes, err := blockchain.GetAllNodesNumber()
	if err != nil {
		log.Errorln("error getting all nodes number from blockchain")
		log.Error(err)
		return err
	}

	nodeNumber := models.NetworkNodeNumber{
		BusyNodes: busyNodes.Uint64(),
		AllNodes:  allNodes.Uint64(),
		ActiveNodes: activeNodes.Uint64(),
	}

	if err := config.GetDB().Model(&nodeNumber).Where("id = ?", 1).Assign(nodeNumber).FirstOrCreate(&models.NetworkNodeNumber{}).Error; err != nil {
		log.Errorln("error update NetworkNodeNumber")
		log.Error(err)
		return err
	}

	totalTasks, runningTasks, queuedTasks, err := blockchain.GetAllTasksNumber()
	if err != nil {
		log.Errorln("error getting all tasks number from blockchain")
		log.Error(err)
		return err
	}

	taskNumber := models.NetworkTaskNumber{
		TotalTasks:   totalTasks.Uint64(),
		RunningTasks: runningTasks.Uint64(),
		QueuedTasks:  queuedTasks.Uint64(),
	}

	if err := config.GetDB().Model(&taskNumber).Where("id = ?", 1).Assign(taskNumber).FirstOrCreate(&models.NetworkTaskNumber{}).Error; err != nil {
		log.Errorln("error update NetworkNodeNumber")
		log.Error(err)
		return err
	}

	step := 100
	var totalGFLOPS float64 = 0
	for start := 0; start < int(allNodes.Int64()); start += step {
		end := start + step
		if end > int(allNodes.Int64()) {
			end = int(allNodes.Int64())
		}

		nodeDatas, err := blockchain.GetAllNodesData(start, end)
		if err != nil {
			log.Errorln("error getting all nodes data from blockchain")
			log.Error(err)
			return err
		}

		for _, data := range nodeDatas {
			nodeData := models.NetworkNodeData{
				Address:   data.Address,
				CardModel: data.CardModel,
				VRam:      data.VRam,
				Balance:   models.BigInt{Int: *data.Balance},
			}
			totalGFLOPS += models.GetGPUGFLOPS(data.CardModel)
			if err := config.GetDB().Model(&nodeData).Where("address = ?", nodeData.Address).Assign(nodeData).FirstOrCreate(&models.NetworkNodeData{}).Error; err != nil {
				log.Errorln("error updating NetworkNodeData")
				log.Error(err)
				return err
			}
		}
	}

	networkFLOPS := models.NetworkFLOPS{GFLOPS: totalGFLOPS}
	if err := config.GetDB().Model(&networkFLOPS).Where("id = ?", 1).Assign(networkFLOPS).FirstOrCreate(&models.NetworkFLOPS{}).Error; err != nil {
		log.Errorln("error updating NetworkFLOPS")
		log.Error(err)
		return err
	}
	return nil
}
