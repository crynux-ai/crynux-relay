package inference_tasks

import (
	"h_relay/api/v1/response"
	"h_relay/models"
)

type TaskResponse struct {
	response.Response
	Data models.InferenceTask `json:"data"`
}
