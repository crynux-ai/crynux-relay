package models

import "gorm.io/gorm"

type WorkerCount struct {
	gorm.Model
	WorkerVersion string `json:"worker_version"`
	Count         uint64 `json:"count"`
}


