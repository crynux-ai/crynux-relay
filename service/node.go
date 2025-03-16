package service

import (
	"context"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"
	"errors"
	"math/big"
	"time"

	"gorm.io/gorm"
)

func SetNodeStatusJoin(ctx context.Context, db *gorm.DB, node *models.Node, modelIDs []string) error {
	appConfig := config.GetConfig()

	return db.Transaction(func(tx *gorm.DB) error {
		if err := Transfer(ctx, tx, node.Address, appConfig.Blockchain.Account.Address, &node.StakeAmount.Int); err != nil {
			return err
		}
		node.Status = models.NodeStatusAvailable
		node.JoinTime = time.Now()
		if err := node.Save(ctx, tx); err != nil {
			return err
		}
		var nodeModels []models.NodeModel
		for _, modelID := range modelIDs {
			model := models.NodeModel{NodeAddress: node.Address, ModelID: modelID, InUse: false}
			nodeModels = append(nodeModels, model)
		}
		if err := models.CreateNodeModels(ctx, tx, nodeModels); err != nil {
			return err
		}
		return nil
	})
}

func SetNodeStatusQuit(ctx context.Context, db *gorm.DB, node *models.Node, slashed bool) error {
	appConfig := config.GetConfig()

	err := db.Transaction(func(tx *gorm.DB) error {
		// delete all node local models
		if err := tx.Where("node_address = ?", node.Address).Delete(&models.NodeModel{}).Error; err != nil {
			return err
		}

		if !slashed {
			if err := Transfer(ctx, tx, appConfig.Blockchain.Account.Address, node.Address, &node.StakeAmount.Int); err != nil {
				return err
			}
		}

		if err := node.Update(ctx, tx, map[string]interface{}{
			"status":                     models.NodeStatusQuit,
			"qos_score":                  0,
			"current_task_id_commitment": sql.NullString{Valid: false},
			"stake_amount":               models.BigInt{Int: *big.NewInt(0)},
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func nodeStartTask(ctx context.Context, db *gorm.DB, node *models.Node, taskIDCommitment string, taskModelIDs []string) error {
	if node.Status != models.NodeStatusAvailable || node.CurrentTaskIDCommitment.Valid {
		return errors.New("node is not available")
	}

	newModels := make([]models.NodeModel, 0)
	unusedModels := make([]models.NodeModel, 0)

	localModelSet := make(map[string]models.NodeModel)
	for _, model := range node.Models {
		localModelSet[model.ModelID] = model
	}
	for _, modelID := range taskModelIDs {
		if model, ok := localModelSet[modelID]; !ok {
			newModel := models.NodeModel{NodeAddress: node.Address, ModelID: modelID, InUse: true}
			newModels = append(newModels, newModel)
		} else if !model.InUse {
			model.InUse = true
			newModels = append(newModels, model)
		}
	}
	taskModelIDSet := make(map[string]struct{})
	for _, modelID := range taskModelIDs {
		taskModelIDSet[modelID] = struct{}{}
	}
	for _, model := range node.Models {
		_, ok := taskModelIDSet[model.ModelID]
		if model.InUse && !ok {
			model.InUse = false
			unusedModels = append(unusedModels, model)
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		node.Update(ctx, tx, map[string]interface{}{
			"status":                     models.NodeStatusBusy,
			"current_task_id_commitment": sql.NullString{String: taskIDCommitment, Valid: true},
		})

		for _, model := range newModels {
			model.Save(ctx, tx)
		}
		for _, model := range unusedModels {
			model.Save(ctx, tx)
		}
		return nil
	})
}

func nodeFinishTask(ctx context.Context, db *gorm.DB, node *models.Node) error {
	if !(node.Status == models.NodeStatusBusy || node.Status == models.NodeStatusPendingPause || node.Status == models.NodeStatusPendingQuit) {
		return errors.New("illegal node status")
	}
	kickout, err := shouldKickoutNode(ctx, node)
	if err != nil {
		return err
	}
	if kickout {
		return db.Transaction(func(tx *gorm.DB) error {
			if err := SetNodeStatusQuit(ctx, db, node, false); err != nil {
				return err
			}
			return emitEvent(ctx, db, &models.NodeKickedOutEvent{NodeAddress: node.Address})
		})
	}

	qosScore, err := getNodeTaskQosScore(ctx, node)
	if err != nil {
		return err
	}
	if node.Status == models.NodeStatusBusy {
		return node.Update(ctx, db, map[string]interface{}{
			"status":                     models.NodeStatusAvailable,
			"current_task_id_commitment": sql.NullString{Valid: false},
			"qos_score":                  qosScore,
		})
	} else if node.Status == models.NodeStatusPendingQuit {
		return SetNodeStatusQuit(ctx, db, node, false)
	} else if node.Status == models.NodeStatusPendingPause {
		return node.Update(ctx, db, map[string]interface{}{
			"status":                     models.NodeStatusPaused,
			"current_task_id_commitment": sql.NullString{Valid: false},
			"qos_score":                  qosScore,
		})
	}
	return nil
}

func nodeSlash(ctx context.Context, db *gorm.DB, node *models.Node) error {
	if !(node.Status == models.NodeStatusBusy || node.Status == models.NodeStatusPendingPause || node.Status == models.NodeStatusPendingQuit) {
		return errors.New("illegal node status")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := SetNodeStatusQuit(ctx, db, node, true); err != nil {
			return err
		}
		return emitEvent(ctx, db, &models.NodeSlashedEvent{NodeAddress: node.Address})
	})
}
