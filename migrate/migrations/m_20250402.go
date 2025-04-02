package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20250402(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTask struct {
		TaskID       string        `json:"task_id" gorm:"index"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250402",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "TaskID"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropIndex(&InferenceTask{}, "TaskID"); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
