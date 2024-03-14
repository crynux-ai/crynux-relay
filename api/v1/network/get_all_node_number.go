package network

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/blockchain"
	"math/big"

	"github.com/gin-gonic/gin"
)

type AllNodeNumber struct {
	AllNodes  *big.Int `json:"all_nodes"`
	BusyNodes *big.Int `json:"busy_nodes"`
}

type GetAllNodeNumberResponse struct {
	response.Response
	Data *AllNodeNumber `json:"data"`
}

func GetAllNodeNumber(_ *gin.Context) (*GetAllNodeNumberResponse, error) {

	busyNodes, allNodes, err := blockchain.GetAllNodesNumber()

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetAllNodeNumberResponse{
		Data: &AllNodeNumber{
			AllNodes:  allNodes,
			BusyNodes: busyNodes,
		},
	}, nil
}
