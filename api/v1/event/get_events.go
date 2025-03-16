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
	StartTime        int64   `query:"start_time" description:"Start time" validate:"required"`
	EndTime          *int64  `query:"end_time" description:"End time"`
	EventType        *string `query:"event_type" description:"Event type"`
	NodeAddress      *string `query:"node_address" description:"Node address"`
	TaskIDCommitment *string `query:"task_id_commitment" description:"Task id commitment"`
	Page             int     `query:"page" description:"Page" default:"1"`
	PageSize         int     `query:"page_size" description:"Page size" default:"50"`
}

type GetEventsResponse struct {
	response.Response
	Data []*models.Event `json:"data"`
}

func GetEvents(c *gin.Context, in *GetEventsInput) (*GetEventsResponse, error) {
	startTime := time.Unix(in.StartTime, 0)
	var endTime time.Time
	if in.EndTime != nil {
		endTime = time.Unix(*in.EndTime, 0)
	} else {
		endTime = time.Now()
	}

	dbCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	var events []*models.Event

	event := &models.Event{}
	if in.EventType != nil {
		event.Type = *in.EventType
	}
	if in.NodeAddress != nil {
		event.NodeAddress = *in.NodeAddress
	}
	if in.TaskIDCommitment != nil {
		event.TaskIDCommitment = *in.TaskIDCommitment
	}
	err := config.GetDB().WithContext(dbCtx).
		Model(event).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Where(event).
		Order("created_at").
		Limit(in.PageSize).
		Offset((in.Page - 1) * in.PageSize).
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return &GetEventsResponse{
		Data: events,
	}, nil
}
