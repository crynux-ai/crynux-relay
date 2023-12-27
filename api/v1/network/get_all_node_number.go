package network

import (
	"github.com/gin-gonic/gin"
	"h_relay/api/v1/response"
	"h_relay/blockchain"
	"math/big"
)

type AllNodeNumber struct {
	AllNodes       *big.Int `json:"all_nodes"`
	AvailableNodes *big.Int `json:"available_nodes"`
}

type GetAllNodeNumberResponse struct {
	response.Response
	Data *AllNodeNumber `json:"data"`
}

func GetAllNodeNumber(_ *gin.Context) (*GetAllNodeNumberResponse, error) {

	allNodes, availableNodes, err := blockchain.GetAllNodesNumber()

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetAllNodeNumberResponse{
		Data: &AllNodeNumber{
			AllNodes:       allNodes,
			AvailableNodes: availableNodes,
		},
	}, nil
}
