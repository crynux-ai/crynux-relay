package network

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AllNodeNumber struct {
	AllNodes  uint64 `json:"all_nodes"`
	BusyNodes uint64 `json:"busy_nodes"`
}

type GetAllNodeNumberResponse struct {
	response.Response
	Data *AllNodeNumber `json:"data"`
}

func GetAllNodeNumber(_ *gin.Context) (*GetAllNodeNumberResponse, error) {

	var nodeNumber models.NetworkNodeNumber
	if err := config.GetDB().Model(&models.NetworkNodeNumber{}).First(&nodeNumber).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewExceptionResponse(err)
		}
	}

	return &GetAllNodeNumberResponse{
		Data: &AllNodeNumber{
			AllNodes:  nodeNumber.AllNodes,
			BusyNodes: nodeNumber.BusyNodes,
		},
	}, nil
}
