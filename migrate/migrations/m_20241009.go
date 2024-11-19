package migrations

import (
	"crynux_relay/models"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20241009(db *gorm.DB) *gormigrate.Gormigrate {
	type InferenceTaskStatusLog struct {
		gorm.Model
		InferenceTaskID uint `gorm:"index"`
		InferenceTask   models.InferenceTask
		Status          models.TaskStatus
	}

	type SelectedNodeStatusLog struct {
		gorm.Model
		SelectedNodeID uint `gorm:"index"`
		SelectedNode   models.SelectedNode
		Status         models.NodeStatus 
	}

	type SelectedNode struct {
		Status models.NodeStatus
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20241009",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().AddColumn(&SelectedNode{}, "Status"); err != nil {
					return err
				}
				if err := tx.Migrator().CreateTable(&InferenceTaskStatusLog{}); err != nil {
					return err
				}
				if err := tx.Migrator().CreateTable(&SelectedNodeStatusLog{}); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropTable(&SelectedNodeStatusLog{}); err != nil {
					return err
				}
				if err := tx.Migrator().DropTable(&InferenceTaskStatusLog{}); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&SelectedNode{}, "Status"); err != nil {
					return err
				}

				return nil
			},
		},
	})
}
