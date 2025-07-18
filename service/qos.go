package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"
	"sync"
	"time"
)

const (
	TASK_SCORE_POOL_SIZE uint64 = 3
	NODE_QOS_SCORE_POOL_SIZE uint64 = 50
	KickoutThreshold     uint64 = 2 // if node has 2 tasks timeout in recent 3 tasks, it will be kicked out
)

var (
	TASK_SCORE_REWARDS [3]uint64 = [3]uint64{10, 5, 2}
	nodeQoSScorePool   NodeQosScorePool = NodeQosScorePool{
		pool: make(map[string][]uint64),
	}
)

func getTaskQosScore(order int) uint64 {
	return TASK_SCORE_REWARDS[order]
}

type NodeQosScorePool struct {
	mu sync.RWMutex
	pool map[string][]uint64
}

func getNodeTaskQosScore(ctx context.Context, node *models.Node, qos uint64) (float64, error) {
	nodeQoSScorePool.mu.RLock()
	qosScorePool, ok := nodeQoSScorePool.pool[node.Address]
	nodeQoSScorePool.mu.RUnlock()
	if !ok {
		qosScores, err := getNodeRecentTaskQosScore(ctx, node, int(NODE_QOS_SCORE_POOL_SIZE))
		if err != nil {
			return 0, err
		}
		qosScorePool = qosScores
	}
	qosScorePool = append(qosScorePool, qos)
	if len(qosScorePool) > int(NODE_QOS_SCORE_POOL_SIZE) {
		qosScorePool = qosScorePool[1:]
	}

	nodeQoSScorePool.mu.Lock()
	nodeQoSScorePool.pool[node.Address] = qosScorePool
	nodeQoSScorePool.mu.Unlock()
	var sum uint64 = 0
	for _, score := range qosScorePool {
		sum += score
	}
	return float64(sum) / float64(len(qosScorePool)), nil
}

func getNodeRecentTaskQosScore(ctx context.Context, node *models.Node, n int) ([]uint64, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	type TaskScore struct {
		QOSScore  sql.NullInt64 `json:"qos_score"`
	}

	var tasks []TaskScore
	err := config.GetDB().WithContext(dbCtx).Unscoped().Model(&models.InferenceTask{}).
		Where("qos_score IS NOT NULL").
		Where("selected_node = ?", node.Address).
		Where("start_time > ?", node.JoinTime).
		Order("start_time DESC").
		Limit(n).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return []uint64{}, nil
	}

	scores := make([]uint64, len(tasks))
	for i, task := range tasks {
		scores[len(tasks) - i - 1] = uint64(task.QOSScore.Int64)
	}
	return scores, nil
}

func shouldKickoutNode(ctx context.Context, node *models.Node) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var tasks []models.InferenceTask

	err := config.GetDB().WithContext(dbCtx).Unscoped().Model(&models.InferenceTask{}).
		Where("selected_node = ?", node.Address).
		Order("id DESC").
		Limit(int(TASK_SCORE_POOL_SIZE)).
		Find(&tasks).Error
	if err != nil {
		return false, err
	}

	timeoutCount := 0
	for _, task := range tasks {
		if task.Status == models.TaskEndAborted && task.AbortReason == models.TaskAbortTimeout {
			timeoutCount++
		}
	}
	if timeoutCount >= int(KickoutThreshold) {
		return true, nil
	}
	return false, nil
}
