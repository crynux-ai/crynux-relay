package incentive

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"time"

	"github.com/gin-gonic/gin"
)

type GetIncentiveOutput struct {
	response.Response
	Data float64 `json:"data"`
}

func GetTodayIncentive(_ *gin.Context) (*GetIncentiveOutput, error) {
	duration := 24 * time.Hour
	now := time.Now().UTC()
	start := now.Truncate(duration)

	var incentive float64
	if err := config.GetDB().Model(&models.NodeIncentive{}).Select("SUM(incentive) as incentive").Where("time >= ?", start).Scan(&incentive).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetIncentiveOutput{Data: incentive}, nil
}

func GetTotalIncentive(_ *gin.Context) (*GetIncentiveOutput, error) {
	var incentive float64
	if err := config.GetDB().Model(&models.NodeIncentive{}).Select("SUM(incentive) as incentive").Scan(&incentive).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &GetIncentiveOutput{Data: incentive}, nil
}