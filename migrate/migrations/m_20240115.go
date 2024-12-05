package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240115(db *gorm.DB) *gormigrate.Gormigrate {

	type ChainTaskType int

	type InferenceTask struct {
		TaskType      ChainTaskType `json:"task_type"`
		VramLimit     uint64        `json:"vram_limit"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240115",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().AddColumn(&InferenceTask{}, "TaskType"); err != nil {
					return err
				}

				return tx.Migrator().AddColumn(&InferenceTask{}, "VramLimit")
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropColumn(&InferenceTask{}, "TaskType"); err != nil {
					return err
				}

				return tx.Migrator().DropColumn(&InferenceTask{}, "VramLimit")
			},
		},
	})
}
