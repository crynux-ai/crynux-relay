package models

import "gorm.io/gorm"

type SyncedBlock struct {
	gorm.Model
	BlockNumber uint64
}
