package nodes

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GetNodeTaskInput struct {
	Address string `json:"address" path:"address" description:"node address"`
}

type GetNodeTaskResponse struct {
	response.Response
	Data string `json:"data" description:"node current task taskIDCommitment, empty string means no task"`
}

const zeroTaskIDCommitment = "0x0000000000000000000000000000000000000000000000000000000000000000"

func GetNodeTask(c *gin.Context, in *GetNodeTaskInput) (*GetNodeTaskResponse, error) {
	node, err := models.GetNodeByAddress(c.Request.Context(), config.GetDB(), in.Address)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &GetNodeTaskResponse{Data: zeroTaskIDCommitment}, nil
	}
	if err != nil {
		return nil, err
	}
	resp := &GetNodeTaskResponse{}
	if node.CurrentTaskIDCommitment.Valid {
		resp.Data = node.CurrentTaskIDCommitment.String
	} else {
		resp.Data = zeroTaskIDCommitment
	}
	return resp, nil
}
