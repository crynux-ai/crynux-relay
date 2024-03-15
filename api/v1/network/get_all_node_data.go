package network

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/blockchain"

	"github.com/gin-gonic/gin"
)

type GetAllNodesDataParams struct {
	Start int `query:"start" json:"start" validate:"min=0"`
	Total int `query:"total" json:"total" validate:"required,max=100,min=1"`
}

type GetAllNodesDataResponse struct {
	response.Response
	Data []blockchain.NodeData `json:"data"`
}

func GetAllNodeData(_ *gin.Context, in *GetAllNodesDataParams) (*GetAllNodesDataResponse, error) {

	allNodeData, err := blockchain.GetAllNodesData(in.Start, in.Start+in.Total)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetAllNodesDataResponse{
		Data: allNodeData,
	}, nil
}
