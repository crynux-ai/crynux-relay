package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20250725(db *gorm.DB) *gormigrate.Gormigrate {

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250725",
			Migrate: func(tx *gorm.DB) error {
				type NetworkNodeData struct {
					gorm.Model
					Address   string  `json:"address" gorm:"index:idx_address_unique,unique"`
					Staking   string  `json:"staking" gorm:"type:string;size:255"`
				}

				return tx.AutoMigrate(&NetworkNodeData{})
			},
			Rollback: func(tx *gorm.DB) error {
				type NetworkNodeData struct {
					gorm.Model
					Address   string  `json:"address" gorm:"index:idx_address_unique,unique"`
					Staking   string  `json:"staking" gorm:"type:string;size:255"`
				}

				if tx.Migrator().HasIndex(&NetworkNodeData{}, "idx_address_unique") {
					return tx.Migrator().DropIndex(&NetworkNodeData{}, "idx_address_unique")
				}

				if tx.Migrator().HasColumn(&NetworkNodeData{}, "Staking") {
					return tx.Migrator().DropColumn(&NetworkNodeData{}, "Staking")
				}

				return nil
			},
		},
	})
}
