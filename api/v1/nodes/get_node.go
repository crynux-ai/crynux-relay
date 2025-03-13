package nodes

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"fmt"

	"github.com/gin-gonic/gin"
)

type GetNodeInput struct {
	Address string `json:"address" path:"address" description:"node address" validate:"required"`
}

func GetNode(c *gin.Context, input *GetNodeInput) (*NodeResponse, error) {
	node, err := models.GetNodeByAddress(c.Request.Context(), config.GetDB(), input.Address)
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	nodeModels, err := models.GetNodeModelsByNodeAddress(c.Request.Context(), config.GetDB(), node.Address)
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	modelIDs := make([]string, 0)
	inUseModelIDs := make([]string, 0)
	for _, model := range nodeModels {
		modelIDs = append(modelIDs, model.ModelID)
		if model.InUse {
			inUseModelIDs = append(inUseModelIDs, model.ModelID)
		}
	}

	nodeVersion := fmt.Sprintf("%d.%d.%d", node.MajorVersion, node.MinorVersion, node.PatchVersion)

	return &NodeResponse{
		Data: &Node{
			Address:       node.Address,
			Status:        node.Status,
			GPUName:       node.GPUName,
			GPUVram:       node.GPUVram,
			QOSScore:      node.QOSScore,
			Version:       nodeVersion,
			InUseModelIDs: inUseModelIDs,
			ModelIDs:      modelIDs,
		},
	}, nil
}
