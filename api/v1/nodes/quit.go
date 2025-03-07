package nodes

import "github.com/gin-gonic/gin"

type QuitInputWithSignature struct {
	Address   string   `json:"address" path:"address" description:"address" validate:"required"`
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func NodeQuit(_ *gin.Context, input *QuitInputWithSignature) (*NodeResponse, error) {
	return nil, nil
}
