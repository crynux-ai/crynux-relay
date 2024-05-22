package migrations

import (
	"crynux_relay/models"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240522(db *gorm.DB) *gormigrate.Gormigrate {
	type NetworkNodeNumber struct {
		gorm.Model
		AllNodes  uint64 `json:"all_nodes"`
		BusyNodes uint64 `json:"busy_nodes"`
	}

	type NetworkTaskNumber struct {
		gorm.Model
		TotalTasks   uint64 `json:"total_tasks"`
		RunningTasks uint64 `json:"running_tasks"`
		QueuedTasks  uint64 `json:"queued_tasks"`
	}

	type NetworkNodeData struct {
		gorm.Model
		Address   string        `json:"address" gorm:"index"`
		CardModel string        `json:"card_model"`
		VRam      int           `json:"v_ram"`
		Balance   models.BigInt `json:"balance" gorm:"type:string;size:255"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240522",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().CreateTable(&NetworkNodeNumber{}); err != nil {
					return err
				}
				if err := tx.Migrator().CreateTable(&NetworkTaskNumber{}); err != nil {
					return err
				}
				if err := tx.Migrator().CreateTable(&NetworkNodeData{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropTable(&NetworkNodeData{}); err != nil {
					return err
				}
				if err := tx.Migrator().DropTable(&NetworkTaskNumber{}); err != nil {
					return err
				}
				if err := tx.Migrator().DropTable(&NetworkNodeNumber{}); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
