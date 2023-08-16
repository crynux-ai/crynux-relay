package inference_tasks

import (
	"errors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/vincent-petithory/dataurl"
	"gorm.io/gorm"
	"h_relay/api/v1/response"
	"h_relay/config"
	"h_relay/models"
	"os"
	"path"
	"strconv"
)

type TaskInput struct {
	TaskId     uint64            `json:"task_id" description:"Task id" validate:"required"`
	Prompt     string            `json:"prompt" validate:"required"`
	BaseModel  string            `json:"base_model" validate:"required"`
	LoraModel  string            `json:"lora_model" default:""`
	TaskConfig models.TaskConfig `json:"task_config"`
	Pose       models.PoseConfig `json:"pose" validate:"required"`
}

type TaskInputWithSignature struct {
	TaskInput
	Timestamp int64  `form:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `form:"signature" json:"signature" description:"Signature" validate:"required"`
}

func CreateTask(_ *gin.Context, in *TaskInputWithSignature) (*TaskResponse, error) {

	match, address, err := ValidateSignature(in.TaskInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	task := models.InferenceTask{
		TaskId: in.TaskId,
	}

	if err := config.GetDB().Where(task).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil,
				response.NewValidationErrorResponse(
					"task_id",
					"Task not found on the Blockchain")
		} else {
			return nil, response.NewExceptionResponse(err)
		}
	}

	if task.Creator != address {
		return nil,
			response.NewValidationErrorResponse(
				"signature",
				"Signer not allowed")
	}

	if task.Status != models.InferenceTaskCreatedOnChain {
		return nil,
			response.NewValidationErrorResponse(
				"task_id",
				"Task already uploaded")
	}

	task.TaskConfig = in.TaskConfig
	task.BaseModel = in.BaseModel
	task.LoraModel = in.LoraModel
	task.Prompt = in.Prompt
	task.PosePreprocess = in.Pose.Preprocess
	task.Status = models.InferenceTaskUploaded

	taskHash, err := task.GetTaskHash()
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if taskHash != task.TaskHash {
		return nil,
			response.NewValidationErrorResponse(
				"task_hash",
				"Task hash mismatch")
	}

	dataHash, err := task.GetDataHash()
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if dataHash != task.DataHash {
		return nil,
			response.NewValidationErrorResponse(
				"data_hash",
				"Data hash mismatch")
	}

	if in.Pose.DataURL != "" {
		dataURL, err := dataurl.DecodeString(in.Pose.DataURL)
		if err != nil {
			return nil, response.NewValidationErrorResponse("pose", "invalid pose image uploaded")
		}

		taskIdStr := strconv.FormatInt(int64(task.TaskId), 100)
		taskFolder := path.Join(config.GetConfig().DataDir.InferenceTasks, taskIdStr)
		poseImgFilename := path.Join(taskFolder, "pose.png")

		poseImgFile, err := os.Create(poseImgFilename)
		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		if dataURL.ContentType() == "image/png" {
			_, err := dataURL.WriteTo(poseImgFile)
			if err != nil {
				return nil, response.NewExceptionResponse(err)
			}
			err = poseImgFile.Close()
			if err != nil {
				return nil, response.NewExceptionResponse(err)
			}
		} else {
			return nil, response.NewValidationErrorResponse("pose", "invalid pose image type")
		}
	}

	if err := config.GetDB().Save(&task).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &TaskResponse{Data: task}, nil
}
