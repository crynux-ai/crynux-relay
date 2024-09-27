package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240927(db *gorm.DB) *gormigrate.Gormigrate {
	type NetworkNodeData struct {
		gorm.Model
		QoS       int64  `json:"qos"`
	}
		return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240927",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().AddColumn(&NetworkNodeData{}, "QoS"); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropColumn(&NetworkNodeData{}, "QoS"); err != nil {
					return err
				}

				return nil
			},
		},
	})
}
