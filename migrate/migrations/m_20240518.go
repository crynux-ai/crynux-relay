package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240518(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTask struct {
		TaskId uint64 `json:"task_id" gorm:"index"`
	}

	type SelectedNode struct {
		InferenceTaskID uint `gorm:"index"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240518",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "TaskId"); err != nil {
					return err
				}

				return tx.Migrator().CreateIndex(&SelectedNode{}, "InferenceTaskID")
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropIndex(&InferenceTask{}, "TaskId"); err != nil {
					return err
				}

				return tx.Migrator().DropIndex(&SelectedNode{}, "InferenceTaskID")
			},
		},
	})
}
