package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20230824(db *gorm.DB) *gormigrate.Gormigrate {

	type SelectedNode struct {
		gorm.Model
		InferenceTaskID  uint
		NodeAddress      string
		Result           string
		IsResultSelected bool
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20230824",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().AddColumn(&SelectedNode{}, "IsResultSelected"); err != nil {
					return err
				}

				return tx.Migrator().AddColumn(&SelectedNode{}, "Result")
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropColumn(&SelectedNode{}, "IsResultSelected"); err != nil {
					return err
				}

				return tx.Migrator().DropColumn(&SelectedNode{}, "Result")
			},
		},
	})
}
