package tasks

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func StartSyncNetwork(ctx context.Context) {
	duration := 3 * time.Minute
	ticker := time.NewTicker(duration)

	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			ticker.Stop()
			log.Errorf("SyncNetwork: stop syncing network due to %v", err)
			return
		case <-ticker.C:
			func() {
				ctx1, cancel := context.WithTimeout(ctx, duration)
				defer cancel()
				log.Infof("SyncNetwork: start syncing network")
				if err := SyncNetwork(ctx1); err != nil {
					log.Errorf("SyncNetwork: sync network error %v", err)
				}
				log.Infof("SyncNetwork: end syncing network")
			}()
		}
	}
}

func getNodeData(ctx context.Context, db *gorm.DB, offset, limit int) ([]models.NetworkNodeData, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var nodes []models.Node
	if err := db.WithContext(dbCtx).Model(&models.Node{}).InnerJoins("Balance").Order("id").Offset(offset).Limit(limit).Find(&nodes).Error; err != nil {
		return nil, err
	}

	var res []models.NetworkNodeData
	for _, node := range nodes {
		res = append(res, models.NetworkNodeData{
			Address:   node.Address,
			CardModel: node.GPUName,
			VRam:      int(node.GPUVram),
			Balance:   node.Balance.Balance,
			QoS:       service.CalculateQosScore(node.QOSScore, service.GetMaxQosScore()),
		})
	}
	return res, nil
}

func syncNodeNumber(ctx context.Context) error {
	busyNodes, err := models.GetBusyNodeCount(ctx, config.GetDB())
	if err != nil {
		log.Errorf("SyncNetwork: error getting busy nodes count %v", err)
		return err
	}
	allNodes, err := models.GetAllNodeCount(ctx, config.GetDB())
	if err != nil {
		log.Errorf("SyncNetwork: error getting all nodes count %v", err)
		return err
	}
	activeNodes, err := models.GetActiveNodeCount(ctx, config.GetDB())
	if err != nil {
		log.Errorf("SyncNetwork: error getting active nodes count %v", err)
		return err
	}

	nodeNumber := models.NetworkNodeNumber{
		BusyNodes:   uint64(busyNodes),
		AllNodes:    uint64(allNodes),
		ActiveNodes: uint64(activeNodes),
	}

	if err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&nodeNumber).Where("id = ?", 1).Assign(nodeNumber).FirstOrCreate(&models.NetworkNodeNumber{}).Error
	}(); err != nil {
		log.Errorf("SyncNetwork: error update NetworkNodeNumber %v", err)
		return err
	}
	return nil
}

func syncTaskNumber(ctx context.Context) error {
	totalTasks, err := models.GetTotalTaskCount(ctx, config.GetDB())
	if err != nil {
		log.Errorf("SyncNetwork: error getting total task count %v", err)
		return err
	}
	runningTasks, err := models.GetRunningTaskCount(ctx, config.GetDB())
	if err != nil {
		log.Errorf("SyncNetwork: error getting running task count %v", err)
		return err
	}
	queuedTasks, err := models.GetQueuedTaskCount(ctx, config.GetDB())
	if err != nil {
		log.Errorf("SyncNetwork: error getting queued task count %v", err)
		return err
	}

	taskNumber := models.NetworkTaskNumber{
		TotalTasks:   uint64(totalTasks),
		RunningTasks: uint64(runningTasks),
		QueuedTasks:  uint64(queuedTasks),
	}

	if err := func() error {
		dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return config.GetDB().WithContext(dbCtx).Model(&taskNumber).Where("id = ?", 1).Assign(taskNumber).FirstOrCreate(&models.NetworkTaskNumber{}).Error
	}(); err != nil {
		log.Errorf("SyncNetwork: error update NetworkNodeNumber")
		log.Error(err)
		return err
	}
	return nil
}

func syncNodeData(ctx context.Context) error {
	limit := 100
	offset := 0
	var totalGFLOPS float64 = 0
	for {
		nodeDatas, err := getNodeData(ctx, config.GetDB(), offset, limit)
		if err != nil {
			log.Errorf("SyncNetwork: error getting nodes data %v", err)
			return err
		}

		for _, data := range nodeDatas {
			totalGFLOPS += models.GetGPUGFLOPS(data.CardModel)
			if err := config.GetDB().WithContext(ctx).Model(&data).Where("address = ?", data.Address).Assign(data).FirstOrCreate(&models.NetworkNodeData{}).Error; err != nil {
				log.Errorf("SyncNetwork: error updating NetworkNodeData %v", err)
				return err
			}
		}
		if len(nodeDatas) == 0 {
			break
		}
		offset += limit
	}

	networkFLOPS := models.NetworkFLOPS{GFLOPS: totalGFLOPS}
	if err := config.GetDB().WithContext(ctx).Model(&networkFLOPS).Where("id = ?", 1).Assign(networkFLOPS).FirstOrCreate(&models.NetworkFLOPS{}).Error; err != nil {
		log.Errorf("SyncNetwork: error updating NetworkFLOPS %v", err)
		return err
	}
	return nil
}

func SyncNetwork(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	wg.Add(3)

	go func() {
		defer wg.Done()
		err := syncNodeNumber(ctx)
		if err != nil {
			log.Errorf("SyncNetwork: error syncing node number %v", err)
		}
		errChan <- err
	}()

	go func() {
		defer wg.Done()
		err := syncTaskNumber(ctx)
		if err != nil {
			log.Errorf("SyncNetwork: error syncing task number %v", err)
		}
		errChan <- err
	}()

	go func() {
		defer wg.Done()
		err := syncNodeData(ctx)
		if err != nil {
			log.Errorf("SyncNetwork: error syncing node data %v", err)
		}
		errChan <- err
	}()

	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}
