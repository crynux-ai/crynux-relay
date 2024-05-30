package network

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NetworkTFLOPS struct {
	TFLOPS float64 `json:"tflops"`
}

type GetNetworkTFLOPSResponse struct {
	response.Response
	Data *NetworkTFLOPS `json:"data"`
}

func GetNetworkTFLOPS(_ *gin.Context) (*GetNetworkTFLOPSResponse, error) {
	var flops models.NetworkFLOPS
	if err := config.GetDB().Model(&flops).First(&flops).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewExceptionResponse(err)
		}
	}

	return &GetNetworkTFLOPSResponse{
		Data: &NetworkTFLOPS{
			TFLOPS: flops.GFLOPS / 1024,
		},
	}, nil
}
