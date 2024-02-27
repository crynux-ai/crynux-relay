package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/models"
)

type GPTResultResponse struct{
	response.Response
	Data models.GPTTaskResponse `json:"data"`
}