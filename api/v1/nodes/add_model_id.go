package nodes

import "github.com/gin-gonic/gin"

type AddModelIDInput struct {
	Address string `json:"address" path:"address" description:"address" validate:"required"`
	ModelID string `json:"model_id" description:"new local model ID" validate:"required"`
}

type AddModelIDInputWithSignature struct {
	AddModelIDInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func AddModelID(_ *gin.Context, input *AddModelIDInputWithSignature) (*NodeResponse, error) {
	return nil, nil
}
