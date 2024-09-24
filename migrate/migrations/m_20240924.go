package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240924(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTask struct {
		gorm.Model
	}
	
	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240924",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "created_at"); err != nil {
					return err
				}
				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "updated_at"); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropTable(&InferenceTask{}, "updated_at"); err != nil {
					return err
				}
				if err := tx.Migrator().DropTable(&InferenceTask{}, "created_at"); err != nil {
					return err
				}

				return nil
			},
		},
	})
}
