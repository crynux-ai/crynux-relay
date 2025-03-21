package migrations

import (
	"crynux_relay/models"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20241015(db *gorm.DB) *gormigrate.Gormigrate {
	type TaskWaitingTimeCount struct {
		gorm.Model

		Start    time.Time       `json:"start"`
		End      time.Time       `json:"end"`
		TaskType models.TaskType `json:"task_type"`
		Seconds  int64           `json:"seconds"`
		Count    int64           `json:"count"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20241015",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().CreateTable(&TaskWaitingTimeCount{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropTable(&TaskWaitingTimeCount{}); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
