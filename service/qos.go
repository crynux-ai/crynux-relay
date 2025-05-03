package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"
	"time"

	"gorm.io/hints"
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
		StartTime sql.NullTime `json:"start_time"`
	}

	var tasks []TaskScore
	err := config.GetDB().WithContext(dbCtx).Unscoped().Model(&models.InferenceTask{}).
		Clauses(hints.UseIndex("idx_inference_tasks_selected_node")).
		Where("selected_node = ?", node.Address).
		Order("id DESC").
		Limit(n).
		Find(&tasks).Error
	if err != nil {
		return 0, 0, err
	}

	var qosScore uint64 = 0
	var taskCount uint64 = 0
	for _, task := range tasks {
		if task.StartTime.Valid && task.StartTime.Time.Before(node.JoinTime) {
			qosScore += task.QOSScore
			taskCount++
		}
	}
	return qosScore, taskCount, nil
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
