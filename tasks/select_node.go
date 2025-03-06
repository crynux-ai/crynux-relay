package tasks

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func selectAllQueuedTasks(ctx context.Context) ([]models.InferenceTask, error) {
	allTasks := make([]models.InferenceTask, 0)

	offset := 0
	limit := 100

	for {
		tasks, err := func(ctx context.Context, offset, limit int) ([]models.InferenceTask, error) {
			tasks := make([]models.InferenceTask, 0)

			dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
				Where("status = ?", models.TaskQueued).
				Order("id").
				Offset(offset).
				Limit(limit).
				Find(&tasks).Error
			if err != nil {
				return nil, err
			}
			return tasks, nil
		}(ctx, offset, limit)
		if err != nil {
			return nil, err
		}
		allTasks = append(allTasks, tasks...)
		if len(tasks) < limit {
			break
		}
		offset += limit
	}
	return allTasks, nil
}

func filterNodesByGPU(ctx context.Context, gpuName string, gpuVram uint64, taskVersion string) ([]models.Node, error) {
	taskVersions := strings.Split(taskVersion, ".")
	if len(taskVersions) != 3 {
		return nil, models.TaskVersionInvalidError
	}
	taskMajorVersion, err := strconv.ParseUint(taskVersions[0], 10, 64)
	if err != nil {
		return nil, models.TaskVersionInvalidError
	}
	taskMinorVersion, err := strconv.ParseUint(taskVersions[1], 10, 64)
	if err != nil {
		return nil, models.TaskVersionInvalidError
	}
	taskPatchVersion, err := strconv.ParseUint(taskVersions[2], 10, 64)
	if err != nil {
		return nil, models.TaskVersionInvalidError
	}

	allNodes := make([]models.Node, 0)

	offset := 0
	limit := 0

	for {
		nodes, err := func (ctx context.Context, offset, limit int) ([]models.Node, error) {
			nodes := make([]models.Node, 0)
			dbCtx, cancel := context.WithTimeout(ctx, 3 * time.Second)
			defer cancel()
		
			err := config.GetDB().WithContext(dbCtx).Model(&models.Node{}).
				Where(&models.Node{Status: models.NodeStatusAvailable, GPUName: gpuName, GPUVram: gpuVram, MajorVersion: taskMajorVersion}).
				Where("minor_version > ? or (minor_version = ? and patch_version > ?)", taskMinorVersion, taskMinorVersion, taskPatchVersion).
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

func filterNodesByVram(ctx context.Context, minVram uint64, taskVersion string) ([]models.Node, error) {
	taskVersions := strings.Split(taskVersion, ".")
	if len(taskVersions) != 3 {
		return nil, models.TaskVersionInvalidError
	}
	taskMajorVersion, err := strconv.ParseUint(taskVersions[0], 10, 64)
	if err != nil {
		return nil, models.TaskVersionInvalidError
	}
	taskMinorVersion, err := strconv.ParseUint(taskVersions[1], 10, 64)
	if err != nil {
		return nil, models.TaskVersionInvalidError
	}
	taskPatchVersion, err := strconv.ParseUint(taskVersions[2], 10, 64)
	if err != nil {
		return nil, models.TaskVersionInvalidError
	}

	allNodes := make([]models.Node, 0)

	offset := 0
	limit := 0

	for {
		nodes, err := func (ctx context.Context, offset, limit int) ([]models.Node, error) {
			nodes := make([]models.Node, 0)
			dbCtx, cancel := context.WithTimeout(ctx, 3 * time.Second)
			defer cancel()
		
			err := config.GetDB().WithContext(dbCtx).Model(&models.Node{}).
				Where(&models.Node{Status: models.NodeStatusAvailable, MajorVersion: taskMajorVersion}).
				Where("gpu_vram >= ?", minVram).
				Where("minor_version > ? or (minor_version = ? and patch_version > ?)", taskMinorVersion, taskMinorVersion, taskPatchVersion).
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
	for i := 0; i < len(nodeModelIDs); i++ {
		if (nodeModelIDs[i] != taskModelIDs[i]) {
			return false
		}
	}
	return true
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

func selectNodeForTask(ctx context.Context, task *models.InferenceTask) (error) {
	var nodes []models.Node
	var err error
	if (len(task.RequiredGPU) > 0) {
		nodes, err = filterNodesByGPU(ctx, task.RequiredGPU, task.RequiredGPUVRAM, task.TaskVersion)
		if err != nil {
			return err
		}
	} else {
		nodes, err = filterNodesByVram(ctx, task.MinVRAM, task.TaskVersion)
		if err != nil {
			return err
		}
	}
	changedNodes := make([]models.Node, 0)
	if len(nodes) > 0 {
		for _, node := range nodes {
			changed := false
			// add additional qos score to nodes with local task models
			cnt := matchModels(node.LocalModelIDs, task.ModelIDs)
			if cnt > 0 {
				node.QOSScore *= uint64(cnt)
				changed = true
			}
			// add additional qos score to nodes with the same last models as task models
			if isSameModels(node.LastModelIDs, task.ModelIDs) {
				node.QOSScore *= 2
				changed = true
			}

			if changed {
				changedNodes = append(changedNodes, node)
			}
		}
	}

	if len(changedNodes) > 0 {
		nodes = changedNodes
	}

	node := selectNodesByScore(nodes)
	
}
