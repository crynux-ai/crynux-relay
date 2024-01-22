package network

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/blockchain"
	"math/big"

	"github.com/gin-gonic/gin"
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

	availableNodes, allNodes, err := blockchain.GetAllNodesNumber()

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
