package inference_tasks

import (
	"crynux_relay/models"

	"github.com/gin-gonic/gin"
)

type AbortTaskInput struct {
	TaskIDCommitment string                 `path:"task_id_commitment" json:"task_id_commitment" description:"Task id commitment" validate:"required"`
	AbortReason      models.TaskAbortReason `json:"abort_reason" description:"Task abort reason" validate:"required"`
}

type AbortTaskInputWithSignature struct {
	AbortTaskInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func AbortTask(_ *gin.Context, input *AbortTaskInputWithSignature) (*TaskResponse, error) {
	return nil, nil
}