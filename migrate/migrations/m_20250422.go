package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"

	"gorm.io/gorm"
)

func M20250422(db *gorm.DB) *gormigrate.Gormigrate {

	type InferenceTask struct {
		ModelSwtiched bool `json:"model_swtiched" gorm:"index"`
	}

	type TaskExecutionTimeCount struct {
		ModelSwitched bool `json:"model_switched" gorm:"index"`
	}
	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250422",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().AddColumn(&InferenceTask{}, "ModelSwtiched"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&TaskExecutionTimeCount{}, "ModelSwitched"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropColumn(&InferenceTask{}, "ModelSwtiched"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&TaskExecutionTimeCount{}, "ModelSwitched"); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
