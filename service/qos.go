package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"
)

const (
	TASK_SCORE_POOL_SIZE uint64 = 3
	KickoutThreshold     uint64 = 10
)

var (
	TASK_SCORE_REWARDS [3]uint64 = [3]uint64{10, 9, 6}
)

func getTaskQosScore(order int) uint64 {
	return TASK_SCORE_REWARDS[order]
}

func getNodeTaskQosScore(ctx context.Context, node *models.Node) (uint64, error) {
	score, count, err := getNodeRecentTaskQosScore(ctx, node, 50)
	if err != nil {
		return 0, err
	}
	return score / count, nil
}

func getNodeRecentTaskQosScore(ctx context.Context, node *models.Node, n int) (uint64, uint64, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	type TaskScore struct {
		QOSScore uint64 `json:"qos_score"`
	}

	var tasks []TaskScore
	err := config.GetDB().WithContext(dbCtx).Model(&models.InferenceTask{}).
		Where("selected_node = ?", node.Address).
		Where("start_time >= ?", node.JoinTime).
		Order("start_time DESC").
		Limit(n).
		Find(&tasks).Error
	if err != nil {
		return 0, 0, err
	}

	var qosScore uint64 = 0
	for _, task := range tasks {
		qosScore += task.QOSScore
	}
	return qosScore, uint64(len(tasks)), nil
}

func shouldKickoutNode(ctx context.Context, node *models.Node) (bool, error) {
	qosScore, count, err := getNodeRecentTaskQosScore(ctx, node, 3)
	if err != nil {
		return false, err
	}
	if count < TASK_SCORE_POOL_SIZE {
		qosScore += (TASK_SCORE_POOL_SIZE - count) * TASK_SCORE_REWARDS[0]
	}
	return qosScore <= KickoutThreshold, nil
}
