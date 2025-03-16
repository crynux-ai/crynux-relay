package models

import (
	"gorm.io/gorm"
)

type Balance struct {
	gorm.Model
	Address string `json:"address" gorm:"uniqueIndex"`
	Balance BigInt `json:"balance" gorm:"type:string;size:255"`
}
