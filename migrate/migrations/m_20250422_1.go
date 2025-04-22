package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"

	"gorm.io/gorm"
)

func M20250422_1(db *gorm.DB) *gormigrate.Gormigrate {

	type InferenceTask struct {
		ModelSwtiched bool `json:"model_swtiched" gorm:"index"`
	}

	type TaskExecutionTimeCount struct {
		ModelSwitched bool `json:"model_switched" gorm:"index"`
	}
	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250422_1",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "ModelSwtiched"); err != nil {
					return err
				}
				if err := tx.Migrator().CreateIndex(&TaskExecutionTimeCount{}, "ModelSwitched"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropIndex(&InferenceTask{}, "ModelSwtiched"); err != nil {
					return err
				}
				if err := tx.Migrator().DropIndex(&TaskExecutionTimeCount{}, "ModelSwitched"); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
