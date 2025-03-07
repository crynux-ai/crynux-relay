package tasks

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func generateQueuedTasks(ctx context.Context, taskCh chan<- *models.InferenceTask) error {
	startID := uint(0)
	limit := 100

	for {
		tasks, err := func(ctx context.Context, startID uint, limit int) ([]models.InferenceTask, error) {
			tasks := make([]models.InferenceTask, 0)

			dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
				Where("status = ?", models.TaskQueued).
				Where("id > ?", startID).
				Order("id").
				Limit(limit).
				Find(&tasks).Error
			if err != nil {
				return nil, err
			}
			return tasks, nil
		}(ctx, startID, limit)
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			time.Sleep(2 * time.Second)
			continue
		}
		for _, task := range tasks {
			taskCh <- &task
		}
		startID = tasks[len(tasks)-1].ID
	}
}

func filterNodesByGPU(ctx context.Context, gpuName string, gpuVram uint64, taskVersionNumbers [3]uint64) ([]models.Node, error) {
	allNodes := make([]models.Node, 0)

	offset := 0
	limit := 0

	for {
		nodes, err := func(ctx context.Context, offset, limit int) ([]models.Node, error) {
			nodes := make([]models.Node, 0)
			dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			err := config.GetDB().WithContext(dbCtx).Model(&models.Node{}).
				Preload("Models").
				Where(&models.Node{Status: models.NodeStatusAvailable, GPUName: gpuName, GPUVram: gpuVram, MajorVersion: taskVersionNumbers[0]}).
				Where("minor_version > ? or (minor_version = ? and patch_version > ?)", taskVersionNumbers[1], taskVersionNumbers[1], taskVersionNumbers[2]).
				Order("id").
				Offset(offset).
				Limit(limit).
				Find(&nodes).Error
			if err != nil {
				return nil, err
			}
			return nodes, nil
		}(ctx, offset, limit)
		if err != nil {
			return nil, err
		}
		allNodes = append(allNodes, nodes...)
		if len(nodes) < limit {
			break
		}
		offset += limit
	}
	return allNodes, nil
}

func filterNodesByVram(ctx context.Context, minVram uint64, taskVersionNumbers [3]uint64) ([]models.Node, error) {
	allNodes := make([]models.Node, 0)

	offset := 0
	limit := 0

	for {
		nodes, err := func(ctx context.Context, offset, limit int) ([]models.Node, error) {
			nodes := make([]models.Node, 0)
			dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			err := config.GetDB().WithContext(dbCtx).Model(&models.Node{}).
				Preload("Models").
				Where(&models.Node{Status: models.NodeStatusAvailable, MajorVersion: taskVersionNumbers[0]}).
				Where("gpu_vram >= ?", minVram).
				Where("minor_version > ? or (minor_version = ? and patch_version > ?)", taskVersionNumbers[1], taskVersionNumbers[1], taskVersionNumbers[2]).
				Order("id").
				Offset(offset).
				Limit(limit).
				Find(&nodes).Error
			if err != nil {
				return nil, err
			}
			return nodes, nil
		}(ctx, offset, limit)
		if err != nil {
			return nil, err
		}
		allNodes = append(allNodes, nodes...)
		if len(nodes) < limit {
			break
		}
		offset += limit
	}
	return allNodes, nil
}

func matchModels(nodeModelIDs, taskModelIDs []string) int {
	nodeModelIDSet := make(map[string]struct{})
	for _, nodeModelID := range nodeModelIDs {
		nodeModelIDSet[nodeModelID] = struct{}{}
	}

	cnt := 0
	for _, taskModelID := range taskModelIDs {
		if _, ok := nodeModelIDSet[taskModelID]; ok {
			cnt += 1
		}
	}
	return cnt
}

func isSameModels(nodeModelIDs, taskModelIDs []string) bool {
	if len(nodeModelIDs) != len(taskModelIDs) {
		return false
	}
	return matchModels(nodeModelIDs, taskModelIDs) == len(nodeModelIDs)
}

func selectNodesByScore(nodes []models.Node) models.Node {
	totalScore := 0
	for _, node := range nodes {
		totalScore += int(node.QOSScore)
	}
	n := rand.Intn(totalScore)
	currentScore := 0
	for _, node := range nodes {
		currentScore += int(node.QOSScore)
		if currentScore > n {
			return node
		}
	}
	return nodes[0]
}

func selectNodeForTask(ctx context.Context, task *models.InferenceTask) (*models.Node, error) {
	var nodes []models.Node
	var err error
	taskVersionNumbers := task.VersionNumbers()
	if len(task.RequiredGPU) > 0 {
		nodes, err = filterNodesByGPU(ctx, task.RequiredGPU, task.RequiredGPUVRAM, taskVersionNumbers)
		if err != nil {
			return nil, err
		}
	} else {
		nodes, err = filterNodesByVram(ctx, task.MinVRAM, taskVersionNumbers)
		if err != nil {
			return nil, err
		}
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	changedNodes := make([]models.Node, 0)
	for _, node := range nodes {
		localModelIDs := make([]string, 0)
		inUseModelIDs := make([]string, 0)
		for _, model := range node.Models {
			localModelIDs = append(localModelIDs, model.ModelID)
			if model.InUse {
				inUseModelIDs = append(inUseModelIDs, model.ModelID)
			}
		}

		changed := false
		// add additional qos score to nodes with local task models
		cnt := matchModels(localModelIDs, task.ModelIDs)
		if cnt > 0 {
			node.QOSScore *= uint64(cnt)
			changed = true
		}
		// add additional qos score to nodes with the same last models as task models
		if isSameModels(inUseModelIDs, task.ModelIDs) {
			node.QOSScore *= 2
			changed = true
		}

		if changed {
			changedNodes = append(changedNodes, node)
		}
	}

	if len(changedNodes) > 0 {
		nodes = changedNodes
	}

	node := selectNodesByScore(nodes)
	return &node, nil
}

func startTask(ctx context.Context, task *models.InferenceTask, node *models.Node) error {
	newModels := make([]models.NodeModel, 0)
	unusedModels := make([]models.NodeModel, 0)

	localModelSet := make(map[string]models.NodeModel)
	for _, model := range node.Models {
		localModelSet[model.ModelID] = model
	}
	for _, modelID := range task.ModelIDs {
		if model, ok := localModelSet[modelID]; !ok {
			newModels = append(newModels, models.NodeModel{NodeAddress: node.Address, ModelID: modelID, InUse: true})
		} else if !model.InUse {
			model.InUse = true
			newModels = append(newModels, model)
		}
	}
	taskModelIDSet := make(map[string]struct{})
	for _, modelID := range task.ModelIDs {
		taskModelIDSet[modelID] = struct{}{}
	}
	for _, model := range node.Models {
		_, ok := taskModelIDSet[model.ModelID]
		if model.InUse && !ok {
			model.InUse = false
			unusedModels = append(unusedModels, model)
		}
	}

	err := config.GetDB().Transaction(func(tx *gorm.DB) error {
		dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		task.Update(dbCtx, tx, &models.InferenceTask{
			SelectedNode: node.Address,
			StartTime:    sql.NullTime{Time: time.Now(), Valid: true},
			Status:       models.TaskParametersUploaded,
		})

		node.Update(dbCtx, tx, &models.Node{
			Status: models.NodeStatusBusy,
		})

		for _, model := range newModels {
			model.Save(dbCtx, tx)
		}
		for _, model := range unusedModels {
			model.Save(dbCtx, tx)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func processQueuedTask(ctx context.Context, taskCh chan *models.InferenceTask) error {
	for task := range taskCh {
		selectedNode, err := selectNodeForTask(ctx, task)
		if err != nil {
			return err
		}
		if selectedNode == nil {
			taskCh <- task
		}
		startTask(ctx, task, selectedNode)
	}
	return nil
}

func StartTaskProcesser(ctx context.Context) {
	taskCh := make(chan *models.InferenceTask, 100)

	go func(ctx context.Context, taskCh chan<- *models.InferenceTask) {
		for {
			err := generateQueuedTasks(ctx, taskCh)
			if err == context.DeadlineExceeded || err == context.Canceled {
				close(taskCh)
				return
			}
			if err != nil {
				log.Errorf("StartTask: generate queued tasks error: %v", err)
			}
			time.Sleep(2 * time.Second)
		}
	}(ctx, taskCh)

	for {
		err := processQueuedTask(ctx, taskCh)
		if err == context.DeadlineExceeded || err == context.Canceled {
			return
		}
		if err != nil {
			log.Errorf("StartTask: process queued tasks error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}
}
