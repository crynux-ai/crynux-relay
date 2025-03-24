package event

import (
	"context"
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	"github.com/gin-gonic/gin"
)

type GetCurrentEventIDInput struct {
	EventType        *string `query:"event_type" description:"Event type"`
	NodeAddress      *string `query:"node_address" description:"Node address"`
	TaskIDCommitment *string `query:"task_id_commitment" description:"Task id commitment"`
}

type GetCurrentEventIDResponse struct {
	response.Response
	Data uint `json:"data"`
}

func GetCurrentEventID(c *gin.Context, in *GetCurrentEventIDInput) (*GetCurrentEventIDResponse, error) {
	dbCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	var event models.Event

	stmt := config.GetDB().WithContext(dbCtx).Model(&event)
	if in.EventType != nil {
		stmt.Where("event_type = ?", *in.EventType)
	}
	if in.NodeAddress != nil {
		stmt.Where("node_address = ?", *in.NodeAddress)
	}
	if in.TaskIDCommitment != nil {
		stmt.Where("task_id_commitment = ?", *in.TaskIDCommitment)
	}
	err := stmt.Order("id DESC").First(&event).Error
	if err != nil {
		return nil, err
	}
	return &GetCurrentEventIDResponse{
		Data: event.ID,
	}, nil
}
