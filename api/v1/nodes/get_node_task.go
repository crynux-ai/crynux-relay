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
	NodeAddress string `json:"node_address" path:"node_address" description:"node address"`
}

type GetNodeTaskResponse struct {
	response.Response
	Data string `json:"data" description:"node current task taskIDCommitment, empty string means no task"`
}

func GetNodeTask(c *gin.Context, in *GetNodeTaskInput) (*GetNodeTaskResponse, error) {
	node, err := models.GetNodeByAddress(c.Request.Context(), config.GetDB(), in.NodeAddress)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewValidationErrorResponse("node_address", "Node not found")
		}
		return nil, err
	}
	resp := &GetNodeTaskResponse{}
	if node.CurrentTaskIDCommitment.Valid {
		resp.Data = node.CurrentTaskIDCommitment.String
	} else {
		resp.Data = ""
	}
	return resp, nil
}
