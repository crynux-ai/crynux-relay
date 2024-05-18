package models

import "gorm.io/gorm"

type SelectedNode struct {
	gorm.Model
	InferenceTaskID  uint  `gorm:"index"`
	NodeAddress      string
	Result           string
	IsResultSelected bool
}
