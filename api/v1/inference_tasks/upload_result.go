package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/validate"
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ResultInput struct {
	TaskIDCommitment string `path:"task_id_commitment" json:"task_id_commitment" description:"Task id commitment" validate:"required"`
}

type ResultInputWithSignature struct {
	ResultInput
	Timestamp int64  `form:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `form:"signature" description:"Signature" validate:"required"`
}

func UploadResult(c *gin.Context, in *ResultInputWithSignature) (*response.Response, error) {

	match, address, err := validate.ValidateSignature(in.ResultInput, in.Timestamp, in.Signature)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	if !match {
		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	task, err := models.GetTaskByIDCommitment(c.Request.Context(), config.GetDB(), in.TaskIDCommitment)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("task_id_commitment", "Task not found")
			return nil, validationErr
		} else {
			return nil, response.NewExceptionResponse(err)
		}
	}

	if task.SelectedNode != address {
		return nil, response.NewValidationErrorResponse("Signature", "Signer not allowed")
	}

	if task.Status != models.TaskValidated && task.Status != models.TaskGroupValidated {
		validationErr := response.NewValidationErrorResponse("task_id_commitment", "Task not validated")
		return nil, validationErr
	}
	// Check whether the images are correct
	var uploadedScoreBytes []byte

	form, err := c.MultipartForm()
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	files, ok := form.File["files"]
	if !ok {
		return nil, response.NewValidationErrorResponse("files", "Files is empty")
	}

	for _, file := range files {

		fileObj, err := file.Open()

		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		var hash []byte
		if task.TaskType == models.TaskTypeSD || task.TaskType == models.TaskTypeSDFTLora {
			hash, err = blockchain.GetPHashForImage(fileObj)
		} else {
			hash, err = blockchain.GetHashForGPTResponse(fileObj)
		}

		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		uploadedScoreBytes = append(uploadedScoreBytes, hash...)

		err = fileObj.Close()
		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}
	}

	uploadedScore := hexutil.Encode(uploadedScoreBytes)

	log.Debugln("image compare: submitted score: " + task.Score)
	log.Debugln("image compare: score from the uploaded file: " + uploadedScore)

	if task.Score != uploadedScore {
		validationErr := response.NewValidationErrorResponse("files", "Wrong result files uploaded")
		return nil, validationErr
	}

	appConfig := config.GetConfig()

	taskDir := filepath.Join(appConfig.DataDir.InferenceTasks, task.TaskIDCommitment, "results")
	if err = os.MkdirAll(taskDir, 0o711); err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	var fileExt string
	if task.TaskType == models.TaskTypeSD || task.TaskType == models.TaskTypeSDFTLora {
		fileExt = ".png"
	} else {
		fileExt = ".json"
	}

	for i, file := range files {
		filename := filepath.Join(taskDir, strconv.Itoa(i)+fileExt)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
	}

	// store checkpoint of finetune type task
	if task.TaskType == models.TaskTypeSDFTLora {
		var checkpoint *multipart.FileHeader
		if checkpoints, ok := form.File["checkpoint"]; !ok {
			return nil, response.NewValidationErrorResponse("checkpoint", "Checkpoint not uploaded")
		} else {
			if len(checkpoints) != 1 {
				return nil, response.NewValidationErrorResponse("checkpoint", "More than one checkpoint file")
			}
			checkpoint = checkpoints[0]
		}
		checkpointFilename := filepath.Join(taskDir, "checkpoint.zip")
		if err := c.SaveUploadedFile(checkpoint, checkpointFilename); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
	}

	err = service.SetTaskStatusEndSuccess(c.Request.Context(), config.GetDB(), task)
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	return &response.Response{}, nil
}
