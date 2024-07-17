package worker

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/config"
	"crynux_relay/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WorkerInput struct {
	Version string `path:"version" json:"version" validate:"required" description:"The worker version"`
}

func WorkerJoin(_ *gin.Context, in *WorkerInput) (*response.Response, error) {
	workerCount := models.WorkerCount{
		WorkerVersion: in.Version,
	}

	if err := config.GetDB().Model(&workerCount).Where(&workerCount).FirstOrCreate(&workerCount).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	if err := config.GetDB().Model(&workerCount).Update("count", gorm.Expr("count + ?", 1)).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &response.Response{}, nil
}

func WorkerQuit(_ *gin.Context, in *WorkerInput) (*response.Response, error) {
	workerCount := models.WorkerCount{
		WorkerVersion: in.Version,
	}

	if err := config.GetDB().Model(&workerCount).Where(&workerCount).First(&workerCount).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.NewValidationErrorResponse("version", "No worker of this version")
		}
		return nil, response.NewExceptionResponse(err)
	}

	if err := config.GetDB().Model(&workerCount).Update("count", gorm.Expr("count - ?", 1)).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &response.Response{}, nil
}

type WorkerCount struct {
	Count uint64 `json:"count"`
}

type WorkerCountResponse struct {
	response.Response
	Data *WorkerCount `json:"data"`
}

func GetWorkerCount(_ *gin.Context, in *WorkerInput) (*WorkerCountResponse, error) {
	workerCount := models.WorkerCount{
		WorkerVersion: in.Version,
	}

	if err := config.GetDB().Model(&workerCount).Where(&workerCount).First(&workerCount).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.NewValidationErrorResponse("version", "No worker of this version")
		}
		return nil, response.NewExceptionResponse(err)
	}

	return &WorkerCountResponse{
		Data: &WorkerCount{
			Count: workerCount.Count,
		},
	}, nil
}
