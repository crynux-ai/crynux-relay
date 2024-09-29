package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240929(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTask struct {
		TaskFee       float64       `json:"task_fee"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240929",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().AddColumn(&InferenceTask{}, "TaskFee"); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropColumn(&InferenceTask{}, "TaskFee"); err != nil {
					return err
				}

				return nil
			},
		},
	})
}
