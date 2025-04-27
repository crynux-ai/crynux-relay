package models

import (
	"time"

	"gorm.io/gorm"
)

type Balance struct {
	gorm.Model
	Address string `json:"address" gorm:"uniqueIndex"`
	Balance BigInt `json:"balance" gorm:"type:string;size:255"`
}

type TransferEventStatus int

const (
	TransferEventStatusPending TransferEventStatus = iota
	TransferEventStatusProcessed
)

type TransferEvent struct {
	ID          uint                `gorm:"primarykey"`
	FromAddress string              `gorm:"not null;index"`
	ToAddress   string              `gorm:"not null;index"`
	Amount      BigInt              `gorm:"not null"`
	CreatedAt   time.Time           `gorm:"not null"`
	Status      TransferEventStatus `gorm:"not null;default:0;index"`
}
