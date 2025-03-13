package balance

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/service"
	"math/big"

	"github.com/gin-gonic/gin"
)

type GetBalanceInput struct {
	Address string `path:"address" json:"address" description:"Address of account"`
}

type GetBalanceResponse struct {
	response.Response
	Data *big.Int `json:"data"`
}

func GetBalance(c *gin.Context, in *GetBalanceInput) (*GetBalanceResponse, error) {
	balance, err := service.GetBalance(c.Request.Context(), config.GetDB(), in.Address)
	if err != nil {
		return nil, err
	}
	return &GetBalanceResponse{
		Data: balance,
	}, nil
}
