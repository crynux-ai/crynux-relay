package incentive

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"
	"database/sql"

	"github.com/gin-gonic/gin"
)

type GetIncentiveOutput struct {
	response.Response
	Data float64 `json:"data"`
}

func GetTotalIncentive(_ *gin.Context) (*GetIncentiveOutput, error) {
	var incentive sql.NullFloat64
	if err := config.GetDB().Unscoped().Model(&models.NodeIncentive{}).Select("SUM(incentive) as incentive").Scan(&incentive).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	var data float64
	if incentive.Valid {
		data = incentive.Float64
	}
	return &GetIncentiveOutput{Data: data}, nil
}
