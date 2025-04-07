package network

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"

	"github.com/gin-gonic/gin"
)

type GetAllNodesDataParams struct {
	Start int `query:"start" json:"start" validate:"min=0"`
	Total int `query:"total" json:"total" validate:"required,max=100,min=1"`
}

type NetworkNodeData struct {
	Address   string  `json:"address"`
	CardModel string  `json:"card_model"`
	VRam      int     `json:"v_ram"`
	Balance   float64 `json:"balance"`
	QoS       int64   `json:"qos"`
}

type GetAllNodesDataResponse struct {
	response.Response
	Data []NetworkNodeData `json:"data"`
}

func GetAllNodeData(_ *gin.Context, in *GetAllNodesDataParams) (*GetAllNodesDataResponse, error) {

	var allNodeData []models.NetworkNodeData
	if err := config.GetDB().Model(&models.NetworkNodeData{}).Order("id ASC").Limit(in.Total).Offset(in.Start).Find(&allNodeData).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	var data []NetworkNodeData
	for _, node := range allNodeData {
		balance, _ := utils.WeiToEther(&node.Balance.Int).Float64()
		data = append(data, NetworkNodeData{
			Address:   node.Address,
			CardModel: node.CardModel,
			VRam:      node.VRam,
			Balance:   balance,
			QoS:       node.QoS,
		})
	}
	return &GetAllNodesDataResponse{
		Data: data,
	}, nil
}
