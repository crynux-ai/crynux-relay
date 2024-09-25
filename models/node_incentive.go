package models

import (
	"time"

	"gorm.io/gorm"
)

type NodeIncentive struct {
	gorm.Model
	NodeAddress string `gorm:"index"`
	Incentive   float64
	Time        time.Time `gorm:"index"`
	TaskCount   int64
}
