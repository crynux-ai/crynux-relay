package nodes

import (
	"crynux_relay/api/v2/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetNodeInput struct {
	Address string `json:"address" path:"address" description:"node address" validate:"required"`
}

func GetNode(c *gin.Context, input *GetNodeInput) (*NodeResponse, error) {
	node, err := models.GetNodeByAddress(c.Request.Context(), config.GetDB(), input.Address)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &NodeResponse{
			Data: &Node{
				Address:       input.Address,
				Status:        models.NodeStatusQuit,
				GPUName:       "",
				GPUVram:       0,
				QOSScore:      0,
				StakingScore:  0,
				ProbWeight:    0,
				Version:       "",
				InUseModelIDs: []string{},
				ModelIDs:      []string{},
			},
		}, nil
	}
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

	stakingScore, qosScore, probWeight := service.CalculateSelectingProb(&node.StakeAmount.Int, service.GetMaxStaking(), node.QOSScore, service.GetMaxQosScore())

	return &NodeResponse{
		Data: &Node{
			Address:       node.Address,
			Status:        node.Status,
			GPUName:       node.GPUName,
			GPUVram:       node.GPUVram,
			QOSScore:      qosScore,
			StakingScore:  stakingScore,
			ProbWeight:    probWeight,
			Version:       nodeVersion,
			InUseModelIDs: inUseModelIDs,
			ModelIDs:      modelIDs,
		},
	}, nil
}
