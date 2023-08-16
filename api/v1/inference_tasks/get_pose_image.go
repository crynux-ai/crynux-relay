package inference_tasks

import "github.com/gin-gonic/gin"

type GetPoseImageInput struct {
	TaskId int64 `path:"task_id" json:"task_id" validate:"required" description:"The task id"`
}

type GetPoseImageInputWithSignature struct {
	GetPoseImageInput
	Timestamp int64  `query:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `query:"signature" json:"signature" description:"Signature" validate:"required"`
}

func GetPoseImage(_ *gin.Context, in *GetPoseImageInputWithSignature) (*TaskResponse, error) {
	return nil, nil
}
