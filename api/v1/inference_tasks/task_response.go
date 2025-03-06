package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/models"
)

type TaskResponse struct {
	response.Response
	Data models.InferenceTask `json:"data"`
}

type TasksResponse struct {
	response.Response
	Data []models.InferenceTask `json:"data"`
}
