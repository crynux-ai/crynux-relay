package balance

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"
	"errors"
	"math/big"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetBalanceInput struct {
	Address string `path:"address" json:"address" description:"Address of account"`
}

type GetBalanceResponse struct {
	response.Response
	Data models.BigInt `json:"data"`
}

func GetBalance(c *gin.Context, in *GetBalanceInput) (*GetBalanceResponse, error) {
	balance, err := service.GetBalance(c.Request.Context(), config.GetDB(), in.Address)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &GetBalanceResponse{
			Data: models.BigInt{Int: *big.NewInt(0)},
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &GetBalanceResponse{
		Data: models.BigInt{Int: *balance},
	}, nil
}
