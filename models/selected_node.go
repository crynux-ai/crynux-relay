package models

import "gorm.io/gorm"

type NodeStatus int

const (
	NodeStatusPending NodeStatus = iota
	NodeStatusRunning
	NodeStatusCancelled
	NodeStatusSlashed
	NodeStatusSuccess
)

type SelectedNode struct {
	gorm.Model
	InferenceTaskID  uint `gorm:"index"`
	NodeAddress      string
	Result           string
	IsResultSelected bool
	Status           NodeStatus
}

type SelectedNodeStatusLog struct {
	gorm.Model
	SelectedNodeID uint `gorm:"index"`
	SelectedNode   SelectedNode
	Status         NodeStatus
}
