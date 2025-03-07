package nodes

import "github.com/gin-gonic/gin"

type UpdateVersionInput struct {
	Address   string   `json:"address" path:"address" description:"address" validate:"required"`
	Version string `json:"version" description:"new node version" validate:"required"`
}

type UpdateVersionInputWithSignature struct {
	UpdateVersionInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func UpdateNodeVersion(_ *gin.Context, input *UpdateVersionInputWithSignature) (*NodeResponse, error) {
	return nil, nil
}