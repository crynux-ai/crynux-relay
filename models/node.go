package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
)

type NodeStatus uint8

const (
	NodeStatusQuit = iota
	NodeStatusAvailable
	NodeStatusBusy
	NodeStatusPendingPause
	NodeStatusPendingQuit
	NodeStatusPaused
)

type Node struct {
	gorm.Model
	Address                 string         `json:"address" gorm:"index"`
	Status                  NodeStatus     `json:"status" gorm:"index"`
	GPUName                 string         `json:"gpu_name" gorm:"index"`
	GPUVram                 uint64         `json:"gpu_vram" gorm:"index"`
	QOSScore                uint64         `json:"qos_score"`
	MajorVersion            uint64         `json:"major_version"`
	MinorVersion            uint64         `json:"minor_version"`
	PatchVersion            uint64         `json:"patch_version"`
	CurrentTaskIDCommitment sql.NullString `json:"current_task_id_commitment" gorm:"null;default:null"`
	CurrentTask             InferenceTask  `json:"-" gorm:"foreignKey:TaskIDCommitment;references:CurrentTaskIDCommitment"`
	Models                  []NodeModel    `json:"-" gorm:"foreignKey:NodeAddress;references:Address"`
}


func (node *Node) Save(ctx context.Context, db *gorm.DB) error {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := db.WithContext(dbCtx).Save(&node).Error; err != nil {
		return err
	}
	return nil
}

func (node *Node) Update(ctx context.Context, db *gorm.DB, newNode *Node) error {
	if node.ID == 0 {
		return errors.New("Node.ID cannot be 0 when update")
	}
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := db.WithContext(dbCtx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(node).Updates(newNode).Error; err != nil {
			return err
		}
		if err := tx.Model(node).First(node).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}


type NodeModel struct {
	gorm.Model
	NodeAddress string `json:"node_address" gorm:"index"`
	ModelID     string `json:"model_id" gorm:"index"`
	InUse       bool   `json:"in_use"`
}


func (nodeModel *NodeModel) Save(ctx context.Context, db *gorm.DB) error {
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := db.WithContext(dbCtx).Save(&nodeModel).Error; err != nil {
		return err
	}
	return nil
}

func (nodeModel *NodeModel) Update(ctx context.Context, db *gorm.DB, newNodeModel *NodeModel) error {
	if nodeModel.ID == 0 {
		return errors.New("Node.ID cannot be 0 when update")
	}
	dbCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := db.WithContext(dbCtx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(nodeModel).Updates(newNodeModel).Error; err != nil {
			return err
		}
		if err := tx.Model(nodeModel).First(nodeModel).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}