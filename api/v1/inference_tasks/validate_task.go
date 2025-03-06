package inference_tasks

import "github.com/gin-gonic/gin"

type ValidateTaskInput struct {
	TaskIDCommitments []string `json:"task_id_commitments" description:"task_id_commitments" validate:"required"`
	TaskID            string   `json:"task_id" description:"task_id" validate:"required"`
	VrfProof          string   `json:"vrf_proof" description:"vrf_proof" validate:"required"`
	PublicKey         string   `json:"public_key" description:"public_key" validate:"required"`
}

type ValidateTaskInputWithSignature struct {
	ValidateTaskInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func ValidateTask(_ *gin.Context, input *ValidateTaskInputWithSignature) (*TasksResponse, error) {
	return nil, nil
}
