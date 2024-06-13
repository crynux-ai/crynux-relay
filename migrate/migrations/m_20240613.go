package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240613(db *gorm.DB) *gormigrate.Gormigrate {
	type NetworkNodeNumber struct {
		ActiveNodes uint64 `json:"active_nodes"`
	}
		
	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240613",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().AddColumn(&NetworkNodeNumber{}, "ActiveNodes"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropColumn(&NetworkNodeNumber{}, "ActiveNodes"); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
