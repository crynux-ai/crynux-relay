package migrations

import (
	"crynux_relay/models"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240925_2(db *gorm.DB) *gormigrate.Gormigrate {
	type TaskCount struct {
		gorm.Model

		Start        time.Time       `json:"start" gorm:"index"`
		End          time.Time       `json:"end"`
		TaskType     models.TaskType `json:"task_type" gorm:"index"`
		TotalCount   int64           `json:"total_count"`
		SuccessCount int64           `json:"success_count"`
		AbortedCount int64           `json:"aborted_count"`
	}

	type TaskExecutionTimeCount struct {
		gorm.Model

		Start    time.Time       `json:"start" gorm:"index"`
		End      time.Time       `json:"end"`
		TaskType models.TaskType `json:"task_type" gorm:"index"`
		Seconds  int64           `json:"seconds"`
		Count    int64           `json:"count"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240925_2",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropIndex(&TaskExecutionTimeCount{}, "TaskType"); err != nil {
					return err
				}
				if err := tx.Migrator().DropIndex(&TaskCount{}, "TaskType"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().CreateIndex(&TaskCount{}, "TaskType"); err != nil {
					return err
				}
				if err := tx.Migrator().CreateIndex(&TaskExecutionTimeCount{}, "TaskType"); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
