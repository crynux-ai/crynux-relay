package balance

import (
	"crynux_relay/api/v1/response"
	"math/big"

	"github.com/gin-gonic/gin"
)

type GetBalanceInput struct {
	Address string `path:"address" json:"address" description:"Address of account"`
}

type GetBalanceInputWithSignature struct {
	GetBalanceInput
	Timestamp int64  `query:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" json:"signature" description:"Signature" validate:"required"`
}

type GetBalanceOutput struct {
	Balance *big.Int `json:"balance"`
}

type GetBalanceResponse struct {
	response.Response
	Data *GetBalanceOutput `json:"data"`
}

func GetBalance(_ *gin.Context, input *GetBalanceInput) (*GetBalanceResponse, error) {
	return nil, nil
}