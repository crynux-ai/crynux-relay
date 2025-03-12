package balance

import (
	"crynux_relay/api/v1/response"
	"math/big"

	"github.com/gin-gonic/gin"
)

type TransferInput struct {
	From  string   `path:"from" json:"from" description:"The from address of the transfer request"`
	To    string   `json:"to" description:"The to address of the transfer request" validate:"required"`
	Value *big.Int `json:"value" description:"The transferred value" validate:"required"`
}

type TransferInputWithSignature struct {
	TransferInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func Transfer(_ *gin.Context, input *TransferInputWithSignature) (*response.Response, error) {
	return nil, nil
}
