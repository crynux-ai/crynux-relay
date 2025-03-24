package event

import (
	"context"
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	"github.com/gin-gonic/gin"
)

type GetEventsInput struct {
	Start            uint    `query:"start" description:"start event id of this query"`
	EventType        *string `query:"event_type" description:"Event type"`
	NodeAddress      *string `query:"node_address" description:"Node address"`
	TaskIDCommitment *string `query:"task_id_commitment" description:"Task id commitment"`
	Limit            int     `query:"limit" description:"Event count limit" default:"50"`
}

type GetEventsResponse struct {
	response.Response
	Data []*models.Event `json:"data"`
}

func GetEvents(c *gin.Context, in *GetEventsInput) (*GetEventsResponse, error) {
	dbCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	var events []*models.Event

	stmt := config.GetDB().WithContext(dbCtx).Model(&models.Event{}).Where("id > ?", in.Start)
	if in.EventType != nil {
		stmt.Where("event_type = ?", *in.EventType)
	}
	if in.NodeAddress != nil {
		stmt.Where("node_address = ?", *in.NodeAddress)
	}
	if in.TaskIDCommitment != nil {
		stmt.Where("task_id_commitment = ?", *in.TaskIDCommitment)
	}
	err := stmt.
		Order("id").
		Limit(in.Limit).
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return &GetEventsResponse{
		Data: events,
	}, nil
}
