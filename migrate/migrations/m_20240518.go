package migrations

import (
	"crynux_relay/models"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240518(db *gorm.DB) *gormigrate.Gormigrate {
	

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240518",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().CreateIndex(&models.InferenceTask{}, "TaskId"); err != nil {
					return err
				}

				return tx.Migrator().CreateIndex(&models.SelectedNode{}, "InferenceTaskID")
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropIndex(&models.InferenceTask{}, "TaskId"); err != nil {
					return err
				}

				return tx.Migrator().DropIndex(&models.SelectedNode{}, "InferenceTaskID")
			},
		},
	})
}
