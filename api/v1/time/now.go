package time

import (
	"crynux_relay/api/v1/response"
	"time"

	"github.com/gin-gonic/gin"
)


type NowTimestamp struct {
	Now int64 `json:"now"`
}

type GetNowResponse struct {
	response.Response
	Data *NowTimestamp `json:"data"`
}

func GetNow(_ *gin.Context) (*GetNowResponse, error) {
	now := time.Now().Unix()
	return &GetNowResponse{
		Data: &NowTimestamp{
			Now: now,
		},
	}, nil
}