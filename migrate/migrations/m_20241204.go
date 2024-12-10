package migrations

import (
	"database/sql"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func M20241204(db *gorm.DB) *gormigrate.Gormigrate {
	type TaskStatus uint8
	type ChainTaskType uint8
	type TaskAbortReason uint8
	type TaskError uint8

	type InferenceTask struct {
		ID               uint            `gorm:"primarykey"`
		CreatedAt        time.Time       `gorm:"index"`
		UpdatedAt        time.Time       `gorm:"index"`
		DeletedAt        gorm.DeletedAt  `gorm:"index"`
		TaskArgs         string          `json:"task_args"`
		TaskIDCommitment string          `json:"task_id_commitment" gorm:"index"`
		Creator          string          `json:"creator"`
		Status           TaskStatus      `json:"status"`
		TaskType         ChainTaskType   `json:"task_type" gorm:"index"`
		MinVRAM          uint64          `json:"min_vram"`
		RequiredGPU      string          `json:"required_gpu"`
		RequiredGPUVRAM  uint64          `json:"required_gpu_vram"`
		TaskFee          float64         `json:"task_fee"`
		TaskSize         uint64          `json:"task_size"`
		ModelID          string          `json:"model_id"`
		AbortReason      TaskAbortReason `json:"abort_reason"`
		TaskError        TaskError       `json:"task_error"`
		SelectedNode     string          `json:"selected_node"`
		// time when task is created (get from blockchain)
		CreateTime sql.NullTime `json:"create_time" gorm:"index;null;default:null"`
		// time when relay report task params are uploaded
		StartTime sql.NullTime `json:"start_time" gorm:"index;null;default:null"`
		// time when task score is ready (get from blockchain)
		ScoreReadyTime sql.NullTime `json:"score_ready_time" gorm:"index;null;default:null"`
		// time when relay find that task score is validated
		ValidatedTime sql.NullTime `json:"validated_time" gorm:"index;null;default:null"`
		// time when relay report task results are uploaded
		ResultUploadedTime sql.NullTime `json:"result_uploaded_time" gorm:"index;null;default:null"`
	}

	return gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "M20241204",
			Migrate: func(tx *gorm.DB) error {
				if err := tx.Migrator().RenameTable("inference_tasks", "old_inference_tasks"); err != nil {
					return err
				}
				if err := tx.Migrator().CreateTable(&InferenceTask{}); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropTable(&InferenceTask{}); err != nil {
					return err
				}
				if err := tx.Migrator().RenameTable("old_inference_tasks", "inference_tasks"); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
