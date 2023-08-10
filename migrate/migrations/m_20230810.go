package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"time"
)

func M20230810(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTask struct {
		ID            uint `gorm:"primary_key"`
		CreatedAt     time.Time
		UpdatedAt     time.Time
		DeletedAt     gorm.DeletedAt
		TaskId        int64 `gorm:"uniqueIndex"`
		Creator       string
		TaskParams    string
		SelectedNodes string
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20230810",
			Migrate: func(tx *gorm.DB) error {
				return tx.Migrator().CreateTable(&InferenceTask{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("inference_tasks")
			},
		},
	})
}
