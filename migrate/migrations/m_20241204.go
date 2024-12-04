package migrations

import (
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
		gorm.Model
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
		CreateTime time.Time `json:"create_time" gorm:"index;null;default:null"`
		// time when relay report task params are uploaded
		StartTime time.Time `json:"start_time" gorm:"index;null;default:null"`
		// time when task score is ready (get from blockchain)
		ScoreReadyTime time.Time `json:"score_ready_time" gorm:"index;null;default:null"`
		// time when relay find that task score is validated
		ValidatedTime time.Time `json:"validated_time" gorm:"index;null;default:null"`
		// time when relay report task results are uploaded
		ResultUploadedTime time.Time `json:"result_uploaded_time" gorm:"index;null;default:null"`
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
				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "CreatedAt"); err != nil {
					return err
				}
				if err := tx.Migrator().CreateIndex(&InferenceTask{}, "UpdatedAt"); err != nil {
					return err
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				if err := tx.Migrator().DropTable(&InferenceTask{}); err != nil {
					return err
				}
				return nil
			},
		},
	})
}
