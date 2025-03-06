package inference_tasks

import "github.com/gin-gonic/gin"

type SubmitScoreInput struct {
	TaskIDCommitment string `path:"task_id_commitment" json:"task_id_commitment" description:"Task id commitment" validate:"required"`
	Score string `json:"score" description:"task score" vaidate:"required"`
}

type SubmitScoreInputWithSignature struct {
	SubmitScoreInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func SubmitScore(_ *gin.Context, input *SubmitScoreInputWithSignature) (*TaskResponse, error) {
	return nil, nil
}
