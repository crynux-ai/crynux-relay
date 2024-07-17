package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240717(db *gorm.DB) *gormigrate.Gormigrate {
	type WorkerCount struct {
		gorm.Model
		WorkerVersion string `json:"worker_version" gorm:"index"`
		Count         uint64 `json:"count"`
	}
	
	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240717",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().CreateTable(&WorkerCount{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropTable(&WorkerCount{}); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
