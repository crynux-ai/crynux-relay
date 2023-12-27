package network

import (
	"github.com/gin-gonic/gin"
	"h_relay/api/v1/response"
	"h_relay/blockchain"
)

type GetAllNodesDataParams struct {
	Start int `query:"start" json:"start" validate:"min=0"`
	Total int `query:"total" json:"total" validate:"required,max=20,min=1"`
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
