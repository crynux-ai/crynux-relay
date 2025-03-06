package nodes

import "github.com/gin-gonic/gin"

type GetNodeInput struct {
	Address string `json:"address" path:"address" description:"node address" validate:"required"`
}

type GetNodeInputWithSignature struct {
	GetNodeInput
	Timestamp int64  `query:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" json:"signature" description:"Signature" validate:"required"`
}

func GetNode(_ *gin.Context, input *GetNodeInputWithSignature) (*NodeResponse, error) {
	return nil, nil
}