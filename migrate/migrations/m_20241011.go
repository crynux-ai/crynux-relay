package migrations

import (
	"crynux_relay/models"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20241011(db *gorm.DB) *gormigrate.Gormigrate {
	type TaskUploadResultTimeCount struct {
		gorm.Model

		Start    time.Time       `json:"start" gorm:"index"`
		End      time.Time       `json:"end"`
		TaskType models.TaskType `json:"task_type" gorm:"index"`
		Seconds  int64           `json:"seconds"`
		Count    int64           `json:"count"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20241011",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().CreateTable(&TaskUploadResultTimeCount{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropTable(&TaskUploadResultTimeCount{}); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
