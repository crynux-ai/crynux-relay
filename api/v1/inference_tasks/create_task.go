package inference_tasks

import (
	"github.com/gin-gonic/gin"
	"h_relay/api/v1/response"
	"h_relay/config"
	"h_relay/models"
	"strconv"
)

type TaskInput struct {
	TaskId        int64  `form:"task_id" json:"task_id" description:"Task id" validate:"required"`
	Creator       string `form:"creator" json:"creator" description:"Creator address" validate:"required"`
	TaskParams    string `form:"task_params" json:"task_params" description:"The detailed task params" validate:"required"`
	SelectedNodes string `form:"selected_nodes" json:"selected_nodes" description:"Selected nodes" validate:"required"`
	Signature     string `form:"signature" json:"signature" description:"The signature of the creator" validate:"required"`
}

func CreateTask(ctx *gin.Context, in *TaskInput) (*TaskResponse, error) {

	sigStr := strconv.FormatInt(in.TaskId, 10) + in.TaskParams + in.SelectedNodes

	match, err := ValidateSignature(in.Creator, sigStr, in.Signature)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	if !match {
		validationErr := response.NewValidationErrorResponse()
		validationErr.SetFieldName("signature")
		validationErr.SetFieldMessage("Invalid signature")
		return nil, validationErr
	}

	task := models.InferenceTask{
		TaskId:        in.TaskId,
		Creator:       in.Creator,
		TaskParams:    in.TaskParams,
		SelectedNodes: in.SelectedNodes,
	}

	if result := config.GetDB().Create(&task); result.Error != nil {
		return nil, response.NewExceptionResponse(result.Error)
	}

	return &TaskResponse{Data: task}, nil
}
