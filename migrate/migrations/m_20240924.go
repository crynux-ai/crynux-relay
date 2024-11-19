package migrations

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240924(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTask struct {
		ID        uint      `gorm:"primarykey"`
		CreatedAt time.Time `gorm:"index"`
		UpdatedAt time.Time `gorm:"index"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240924",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "CreatedAt"); err != nil {
					return err
				}
				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "UpdatedAt"); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropIndex(&InferenceTask{}, "UpdatedAt"); err != nil {
					return err
				}
				if err := tx.Migrator().DropIndex(&InferenceTask{}, "CreatedAt"); err != nil {
					return err
				}

				return nil
			},
		},
	})
}
