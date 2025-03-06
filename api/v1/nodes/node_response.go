package nodes

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/models"
)

type NodeResponse struct {
	response.Response
	Data models.Node `json:"data"`
}
