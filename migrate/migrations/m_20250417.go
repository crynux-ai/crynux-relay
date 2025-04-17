package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"

	"gorm.io/gorm"
)

func M20250417(db *gorm.DB) *gormigrate.Gormigrate {

	type InferenceTask struct {
		Status       uint8  `json:"status" gorm:"index"`
		SelectedNode string `json:"selected_node" gorm:"index"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{

		{

			ID: "M20250402",

			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "Status"); err != nil {
					return err
				}
				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "SelectedNode"); err != nil {
					return err
				}
				return nil
			},

			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropIndex(&InferenceTask{}, "SelectedNode"); err != nil {
					return err
				}
				if err := tx.Migrator().DropIndex(&InferenceTask{}, "Status"); err != nil {
					return err
				}
				return nil
			},
		},
	})

}
