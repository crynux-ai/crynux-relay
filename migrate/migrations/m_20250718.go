package migrations

import (
	"database/sql"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20250718(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTask struct {
		QOSScore     sql.NullInt64 `json:"qos_score" gorm:"index:idx_qos_selected_node_start_time,priority:1"`
		SelectedNode string        `json:"selected_node" gorm:"index:idx_qos_selected_node_start_time,priority:2"`
		StartTime    sql.NullTime  `json:"start_time" gorm:"index;null;default:null;index:idx_qos_selected_node_start_time,priority:3"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250718",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&InferenceTask{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropIndex(&InferenceTask{}, "idx_qos_selected_node_start_time")
			},
		},
	})
}
