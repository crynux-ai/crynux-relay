package migrations

import (
	"crynux_relay/models"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20230810(db *gorm.DB) *gormigrate.Gormigrate {

	type SelectedNode struct {
		gorm.Model
		InferenceTaskID uint
		NodeAddress     string
	}

	type SyncedBlock struct {
		gorm.Model
		BlockNumber uint64
	}

	type InferenceTask struct {
		gorm.Model
		TaskArgs      string
		TaskId        uint64
		Creator       string
		TaskHash      string
		DataHash      string
		Status        models.TaskStatus
		SelectedNodes []SelectedNode
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20230810",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().CreateTable(&SelectedNode{}); err != nil {
					return err
				}

				if err := tx.Migrator().CreateTable(&SyncedBlock{}); err != nil {
					return err
				}

				return tx.Migrator().CreateTable(&InferenceTask{})
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropTable("inference_tasks"); err != nil {
					return err
				}
				if err := tx.Migrator().DropTable("selected_nodes"); err != nil {
					return err
				}
				return tx.Migrator().DropTable("synced_blocks")
			},
		},
	})
}
