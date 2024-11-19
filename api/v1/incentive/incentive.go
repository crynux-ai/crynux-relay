package incentive

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"

	"github.com/gin-gonic/gin"
)

type GetIncentiveOutput struct {
	response.Response
	Data float64 `json:"data"`
}

func GetTotalIncentive(_ *gin.Context) (*GetIncentiveOutput, error) {
	var incentive float64
	if err := config.GetDB().Model(&models.NodeIncentive{}).Select("SUM(incentive) as incentive").Scan(&incentive).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetIncentiveOutput{Data: incentive}, nil
}
