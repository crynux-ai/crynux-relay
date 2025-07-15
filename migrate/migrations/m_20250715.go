package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20250715(db *gorm.DB) *gormigrate.Gormigrate {

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250715",
			Migrate: func(tx *gorm.DB) error {
				type Node struct {
					QOSScore float64 `json:"qos_score"`
				}
			
				type NetworkNodeData struct {
					QoS float64 `json:"qos"`
				}
			
				return tx.AutoMigrate(&Node{}, &NetworkNodeData{})
			},
			Rollback: func(tx *gorm.DB) error {
				type Node struct {
					QOSScore uint64 `json:"qos_score"`
				}

				type NetworkNodeData struct {
					QoS int64 `json:"qos"`
				}

				return tx.AutoMigrate(&Node{}, &NetworkNodeData{})
			},
		},
	})
}
