package inference_tasks

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/api/v1/response"
	"h_relay/config"
	"h_relay/models"
)

type TaskInput struct {
	SelectedNodes string `form:"selected_nodes" json:"selected_nodes" description:"Selected nodes" validate:"required"`
	TaskId        int64  `form:"task_id" json:"task_id" description:"Task id" validate:"required"`
	TaskParams    string `form:"task_params" json:"task_params" description:"The detailed task params" validate:"required"`
}

type TaskInputWithSignature struct {
	TaskInput
	Timestamp int64  `form:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `form:"signature" json:"signature" description:"Signature" validate:"required"`
}

func CreateTask(ctx *gin.Context, in *TaskInputWithSignature) (*TaskResponse, error) {

	sigStr, err := json.Marshal(in.TaskInput)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	match, address, err := ValidateSignature(sigStr, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	task := models.InferenceTask{
		TaskId:        in.TaskId,
		Creator:       address,
		TaskParams:    in.TaskParams,
		SelectedNodes: in.SelectedNodes,
	}

	if err := config.GetDB().Create(&task).Error; err != nil {

		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, response.NewValidationErrorResponse("task_id", "Duplicated task")
		}

		return nil, response.NewExceptionResponse(err)
	}

	return &TaskResponse{Data: task}, nil
}
