package inference_tasks

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"h_relay/api/v1/response"
	"h_relay/config"
	"h_relay/models"
	"strconv"
)

func GetTaskById(ctx *gin.Context) (*TaskResponse, error) {
	taskId, err := strconv.Atoi(ctx.Param("task_id"))

	if err != nil {
		validationErr := response.NewValidationErrorResponse()
		validationErr.SetFieldName("task_id")
		validationErr.SetFieldMessage("Invalid task id")

		return nil, validationErr
	}

	var task models.InferenceTask

	if result := config.GetDB().Where(&models.InferenceTask{TaskId: int64(taskId)}).First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse()
			validationErr.SetFieldName("task_id")
			validationErr.SetFieldMessage("Task not found")

			return nil, validationErr
		} else {
			return nil, response.NewExceptionResponse(result.Error)
		}
	}

	return &TaskResponse{Data: task}, nil
}
