package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20230810(db *gorm.DB) *gormigrate.Gormigrate {
	type TaskConfig struct {
		ImageWidth  int     `description:"Image width"`
		ImageHeight int     `description:"Image height"`
		LoraWeight  float32 `description:"Weight of the LoRA model"`
		NumImages   int     `description:"Number of images to generate"`
		Seed        int     `description:"The random seed used to generate images"`
		Steps       int     `description:"Steps"`
	}

	type PoseConfig struct {
		DataURL    string  `description:"The pose image DataURL"`
		PoseWeight float32 `description:"Weight of the pose model"`
		Preprocess bool    `description:"Preprocess the image"`
	}

	type SelectedNode struct {
		gorm.Model
		InferenceTaskID uint
		NodeAddress     string
	}

	type InferenceTask struct {
		gorm.Model
		TaskId        uint64 `gorm:"uniqueIndex"`
		Creator       string
		TaskHash      string
		DataHash      string
		Prompt        string
		BaseModel     string
		LoraModel     string
		TaskConfig    *TaskConfig `gorm:"embedded"`
		Pose          *PoseConfig `gorm:"embedded"`
		Status        int         ``
		SelectedNodes []SelectedNode
	}

	type SyncedBlock struct {
		gorm.Model
		BlockNumber uint64
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
				return tx.Migrator().DropTable("inference_tasks")
			},
		},
	})
}
