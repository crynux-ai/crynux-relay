package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20240530(db *gorm.DB) *gormigrate.Gormigrate {
	type NetworkFLOPS struct {
		gorm.Model
		GFLOPS float64 `json:"gflops"`
	}
	
	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20240530",
			Migrate: func(tx *gorm.DB) error {

				if err := tx.Migrator().CreateTable(&NetworkFLOPS{}); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {

				if err := tx.Migrator().DropTable(&NetworkFLOPS{}); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
