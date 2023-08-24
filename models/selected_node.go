package models

import "gorm.io/gorm"

type SelectedNode struct {
	gorm.Model
	InferenceTaskID uint
	NodeAddress     string
	Result          string
}
