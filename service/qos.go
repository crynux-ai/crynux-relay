package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"
	"time"
)

const (
	TASK_SCORE_POOL_SIZE uint64 = 3
	KickoutThreshold     uint64 = 2 // if node has 2 tasks timeout in recent 3 tasks, it will be kicked out
)

var (
	TASK_SCORE_REWARDS [3]uint64 = [3]uint64{10, 5, 2}
)

func getTaskQosScore(order int) uint64 {
	return TASK_SCORE_REWARDS[order]
}

func getNodeTaskQosScore(ctx context.Context, node *models.Node) (float64, error) {
	score, count, err := getNodeRecentTaskQosScore(ctx, node, 50)
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return float64(TASK_SCORE_REWARDS[0]) / 2, nil
	}
	return float64(score) / float64(count), nil
}

func getNodeRecentTaskQosScore(ctx context.Context, node *models.Node, n int) (uint64, uint64, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	type TaskScore struct {
		ID        uint          `json:"id"`
		QOSScore  sql.NullInt64 `json:"qos_score"`
		StartTime sql.NullTime  `json:"start_time"`
	}

	var tasks []TaskScore
	err := config.GetDB().WithContext(dbCtx).Unscoped().Model(&models.InferenceTask{}).
		Where("selected_node = ?", node.Address).
		Where("start_time > ?", node.JoinTime).
		Where("qos_score IS NOT NULL").
		Order("id DESC").
		Limit(n).
		Find(&tasks).Error
	if err != nil {
		return 0, 0, err
	}

	var qosScore uint64 = 0
	var taskCount uint64 = 0
	for _, task := range tasks {
		qosScore += uint64(task.QOSScore.Int64)
		taskCount++
	}
	return qosScore, taskCount, nil
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
