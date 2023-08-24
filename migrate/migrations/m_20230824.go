package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20230824(db *gorm.DB) *gormigrate.Gormigrate {

	type SelectedNode struct {
		gorm.Model
		InferenceTaskID uint
		NodeAddress     string
		Result          string
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20230824",
			Migrate: func(tx *gorm.DB) error {
				return tx.Migrator().AddColumn(&SelectedNode{}, "Result")
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn(&SelectedNode{}, "Result")
			},
		},
	})
}
