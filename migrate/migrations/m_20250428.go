package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20250428(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTask struct {
		TaskIDCommitment string `json:"task_id_commitment" gorm:"uniqueIndex"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250428",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&InferenceTask{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropIndex(&InferenceTask{}, "task_id_commitment")
			},
		},
	})
}
