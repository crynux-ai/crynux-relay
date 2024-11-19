package migrations

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240925_3(db *gorm.DB) *gormigrate.Gormigrate {
	type NodeIncentive struct {
		gorm.Model
		NodeAddress string        `gorm:"index"`
		Incentive   float64
		Time        time.Time     `gorm:"index"`
		TaskCount   int64
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240925_3",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().CreateTable(&NodeIncentive{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropTable(&NodeIncentive{}); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
