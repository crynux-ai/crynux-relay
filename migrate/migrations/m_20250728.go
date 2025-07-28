package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20250728(db *gorm.DB) *gormigrate.Gormigrate {

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250728",
			Migrate: func(tx *gorm.DB) error {
				type NodeIncentive struct {
					SDTaskCount       int64 `gorm:"column:sd_task_count"`
					LLMTaskCount      int64 `gorm:"column:llm_task_count"`
					SDFTLoraTaskCount int64 `gorm:"column:sd_ft_lora_task_count"`
				}

				return tx.AutoMigrate(&NodeIncentive{})
			},
			Rollback: func(tx *gorm.DB) error {
				type NodeIncentive struct {
					SDTaskCount       int64 `gorm:"column:sd_task_count"`
					LLMTaskCount      int64 `gorm:"column:llm_task_count"`
					SDFTLoraTaskCount int64 `gorm:"column:sd_ft_lora_task_count"`
				}

				if tx.Migrator().HasColumn(&NodeIncentive{}, "sd_task_count") {
					return tx.Migrator().DropColumn(&NodeIncentive{}, "sd_task_count")
				}

				if tx.Migrator().HasColumn(&NodeIncentive{}, "llm_task_count") {
					return tx.Migrator().DropColumn(&NodeIncentive{}, "llm_task_count")
				}

				if tx.Migrator().HasColumn(&NodeIncentive{}, "sd_ft_lora_task_count") {
					return tx.Migrator().DropColumn(&NodeIncentive{}, "sd_ft_lora_task_count")
				}

				return nil
			},
		},
	})
}
