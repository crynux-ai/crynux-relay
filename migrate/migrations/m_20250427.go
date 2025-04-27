package migrations

import (
	"crynux_relay/models"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20250427(db *gorm.DB) *gormigrate.Gormigrate {
	type TransferEvent struct {
		ID          uint          `gorm:"primarykey"`
		FromAddress string        `gorm:"not null;index"`
		ToAddress   string        `gorm:"not null;index"`
		Amount      models.BigInt `gorm:"not null;type:string;size:255"`
		CreatedAt   time.Time     `gorm:"not null"`
		Status      int           `gorm:"not null;default:0;index"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250427",
			Migrate: func(tx *gorm.DB) error {
				return tx.Migrator().CreateTable(&TransferEvent{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable(&TransferEvent{})
			},
		},
	})
}
