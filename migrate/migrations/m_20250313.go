package migrations

import (
	"crynux_relay/models"
	"database/sql"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20250313(db *gorm.DB) *gormigrate.Gormigrate {
	type Balance struct {
		ID        uint           `gorm:"primarykey"`
		CreatedAt time.Time      `gorm:"index"`
		UpdatedAt time.Time      `gorm:"index"`
		DeletedAt gorm.DeletedAt `gorm:"index"`
		Address   string         `json:"address" gorm:"uniqueIndex;type:string;size:255"`
		Balance   models.BigInt  `json:"balance" gorm:"type:string;size:255"`
	}

	type Event struct {
		ID               uint           `gorm:"primarykey"`
		CreatedAt        time.Time      `gorm:"index"`
		UpdatedAt        time.Time      `gorm:"index"`
		DeletedAt        gorm.DeletedAt `gorm:"index"`
		Type             string         `json:"type" gorm:"index"`
		NodeAddress      string         `json:"node_address" gorm:"index"`
		TaskIDCommitment string         `json:"task_id_commitment" gorm:"index"`
		Args             string         `json:"args"`
	}

	type InferenceTask struct {
		SamplingSeed string        `json:"sampling_seed"`
		Nonce        string        `json:"nonce"`
		TaskVersion  string        `json:"task_version"`
		Timeout      uint64        `json:"timeout"`
		Score        string        `json:"score" gorm:"type:text"`
		QOSScore     uint64        `json:"qos_score"`
		TaskID       string        `json:"task_id"`
		TaskFee      models.BigInt `json:"task_fee" gorm:"type:string;size:255"`
	}

	type NodeModel struct {
		ID          uint           `gorm:"primarykey"`
		CreatedAt   time.Time      `gorm:"index"`
		UpdatedAt   time.Time      `gorm:"index"`
		DeletedAt   gorm.DeletedAt `gorm:"index"`
		NodeAddress string         `json:"node_address" gorm:"index"`
		ModelID     string         `json:"model_id" gorm:"index"`
		InUse       bool           `json:"in_use"`
	}

	type Node struct {
		ID                      uint              `gorm:"primarykey"`
		CreatedAt               time.Time         `gorm:"index"`
		UpdatedAt               time.Time         `gorm:"index"`
		DeletedAt               gorm.DeletedAt    `gorm:"index"`
		Address                 string            `json:"address" gorm:"index"`
		Status                  models.NodeStatus `json:"status" gorm:"index"`
		GPUName                 string            `json:"gpu_name" gorm:"index"`
		GPUVram                 uint64            `json:"gpu_vram" gorm:"index"`
		QOSScore                uint64            `json:"qos_score"`
		MajorVersion            uint64            `json:"major_version"`
		MinorVersion            uint64            `json:"minor_version"`
		PatchVersion            uint64            `json:"patch_version"`
		JoinTime                time.Time         `json:"join_time"`
		StakeAmount             models.BigInt     `json:"stake_amount"`
		CurrentTaskIDCommitment sql.NullString    `json:"current_task_id_commitment" gorm:"null;default:null"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20250313",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.AutoMigrate(&Balance{}, &Event{}, &Node{}, &NodeModel{}); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&InferenceTask{}, "SamplingSeed"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&InferenceTask{}, "Nonce"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&InferenceTask{}, "TaskVersion"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&InferenceTask{}, "Timeout"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&InferenceTask{}, "Score"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&InferenceTask{}, "QOSScore"); err != nil {
					return err
				}
				if err := tx.Migrator().AddColumn(&InferenceTask{}, "TaskID"); err != nil {
					return err
				}
				if err := tx.Migrator().AlterColumn(&InferenceTask{}, "TaskFee"); err != nil {
					return err
				}
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropTable(&Balance{}, &Event{}, &Node{}, &NodeModel{}); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&InferenceTask{}, "SamplingSeed"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&InferenceTask{}, "Nonce"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&InferenceTask{}, "TaskVersion"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&InferenceTask{}, "Timeout"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&InferenceTask{}, "Score"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&InferenceTask{}, "QOSScore"); err != nil {
					return err
				}
				if err := tx.Migrator().DropColumn(&InferenceTask{}, "TaskID"); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
