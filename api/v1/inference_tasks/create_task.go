package inference_tasks

import (
	"github.com/gin-gonic/gin"
	"h_relay/api/v1/response"
	"h_relay/config"
	"h_relay/models"
)

type TaskInput struct {
	TaskId     int64  `form:"task_id" json:"task_id" description:"Task id" validate:"required"`
	Creator    string `form:"creator" json:"creator" description:"Creator address" validate:"required"`
	TaskParams string `form:"task_params" json:"task_params" description:"The detailed task params" validate:"required"`
	Signature  string `form:"signature" json:"signature" description:"The signature of the creator" validate:"required"`
}

func CreateTask(ctx *gin.Context, in *TaskInput) (*TaskResponse, error) {

	//TODO: Validate creator signature

	task := models.InferenceTask{
		TaskId:     in.TaskId,
		Creator:    in.Creator,
		TaskParams: in.TaskParams,
	}

	if result := config.GetDB().Create(&task); result.Error != nil {
		return nil, response.NewExceptionResponse(result.Error)
	}

	return &TaskResponse{Data: task}, nil
}
