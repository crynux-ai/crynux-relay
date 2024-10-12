package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20241012(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTask struct {
		gorm.Model
		AbortReason string `json:"abort_reason"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20241012",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().AddColumn(&InferenceTask{}, "AbortReason"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropColumn(&InferenceTask{}, "AbortReason"); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
