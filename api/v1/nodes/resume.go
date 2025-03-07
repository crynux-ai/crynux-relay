package nodes

import "github.com/gin-gonic/gin"

type ResumeInputWithSignature struct {
	Address   string   `json:"address" path:"address" description:"address" validate:"required"`
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func NodeResume(_ *gin.Context, input *ResumeInputWithSignature) (*NodeResponse, error) {
	return nil, nil
}

