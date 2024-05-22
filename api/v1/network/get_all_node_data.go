package network

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"

	"github.com/gin-gonic/gin"
)

type GetAllNodesDataParams struct {
	Start int `query:"start" json:"start" validate:"min=0"`
	Total int `query:"total" json:"total" validate:"required,max=100,min=1"`
}

type GetAllNodesDataResponse struct {
	response.Response
	Data []models.NetworkNodeData `json:"data"`
}

func GetAllNodeData(_ *gin.Context, in *GetAllNodesDataParams) (*GetAllNodesDataResponse, error) {

	var allNodeData []models.NetworkNodeData
	if err := config.GetDB().Model(&models.NetworkNodeData{}).Order("id ASC").Limit(in.Total).Offset(in.Start).Find(&allNodeData).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetAllNodesDataResponse{
		Data: allNodeData,
	}, nil
}
