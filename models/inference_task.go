package models

import (
	"gorm.io/gorm"
	"strconv"
)

type InferenceTask struct {
	gorm.Model
	TaskId        int64  `form:"task_id" json:"task_id" description:"Task id"`
	Creator       string `form:"creator" json:"creator" description:"Creator address"`
	TaskParams    string `form:"task_params" json:"task_params" description:"The detailed task params"`
	SelectedNodes string `form:"selected_nodes" json:"selected_nodes" description:"The selected nodes"`
}

func (it *InferenceTask) GetTaskIdAsString() string {
	return strconv.FormatInt(it.TaskId, 10)
}
