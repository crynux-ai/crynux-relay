package nodes

import "github.com/gin-gonic/gin"

type PauseInputWithSignature struct {
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func NodePause(_ *gin.Context, input *PauseInputWithSignature) (*NodeResponse, error) {
	return nil, nil
}
