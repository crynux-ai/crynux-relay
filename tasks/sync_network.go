package tasks

import (
	"context"
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	log "github.com/sirupsen/logrus"
)

func StartSyncNetwork(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			ticker.Stop()
			log.Errorf("SyncNetwork: stop syncing network due to %v", err)
			return
		case <-ticker.C:
			func() {
				ctx1, cancel := context.WithTimeout(ctx, time.Minute)
				defer cancel()
				if err := SyncNetwork(ctx1); err != nil {
					log.Errorf("SyncNetwork: sync network error %v", err)
				}
			}()
		}
	}
}

func SyncNetwork(ctx context.Context) error {
	busyNodes, allNodes, activeNodes, err := blockchain.GetAllNodesNumber(ctx)
	if err != nil {
		log.Errorln("SyncNetwork: error getting all nodes number from blockchain")
		log.Error(err)
		return err
	}

	nodeNumber := models.NetworkNodeNumber{
		BusyNodes:   busyNodes.Uint64(),
		AllNodes:    allNodes.Uint64(),
		ActiveNodes: activeNodes.Uint64(),
	}

	if err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&nodeNumber).Where("id = ?", 1).Assign(nodeNumber).FirstOrCreate(&models.NetworkNodeNumber{}).Error
	}(); err != nil {
		log.Errorln("SyncNetwork: error update NetworkNodeNumber")
		log.Error(err)
		return err
	}

	totalTasks, runningTasks, queuedTasks, err := blockchain.GetAllTasksNumber(ctx)
	if err != nil {
		log.Errorln("SyncNetwork: error getting all tasks number from blockchain")
		log.Error(err)
		return err
	}

	taskNumber := models.NetworkTaskNumber{
		TotalTasks:   totalTasks.Uint64(),
		RunningTasks: runningTasks.Uint64(),
		QueuedTasks:  queuedTasks.Uint64(),
	}

	if err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&taskNumber).Where("id = ?", 1).Assign(taskNumber).FirstOrCreate(&models.NetworkTaskNumber{}).Error
	}(); err != nil {
		log.Errorln("SyncNetwork: error update NetworkNodeNumber")
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

		nodeDatas, err := blockchain.GetAllNodesData(ctx, start, end)
		if err != nil {
			log.Errorln("SyncNetwork: error getting all nodes data from blockchain")
			log.Error(err)
			return err
		}

		for _, data := range nodeDatas {
			nodeData := models.NetworkNodeData{
				Address:   data.Address,
				CardModel: data.CardModel,
				VRam:      data.VRam,
				Balance:   models.BigInt{Int: *data.Balance},
				QoS:       data.QoS,
			}
			totalGFLOPS += models.GetGPUGFLOPS(data.CardModel)
			if err := func() error {
				dbCtx, cancel := context.WithTimeout(ctx, time.Second)
				defer cancel()
				return config.GetDB().WithContext(dbCtx).Model(&nodeData).Where("address = ?", nodeData.Address).Assign(nodeData).FirstOrCreate(&models.NetworkNodeData{}).Error
			}(); err != nil {
				log.Errorln("SyncNetwork: error updating NetworkNodeData")
				log.Error(err)
				return err
			}
		}
	}

	networkFLOPS := models.NetworkFLOPS{GFLOPS: totalGFLOPS}
	if err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&networkFLOPS).Where("id = ?", 1).Assign(networkFLOPS).FirstOrCreate(&models.NetworkFLOPS{}).Error
	}(); err != nil {
		log.Errorln("SyncNetwork: error updating NetworkFLOPS")
		log.Error(err)
		return err
	}
	return nil
}
