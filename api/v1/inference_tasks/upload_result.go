package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ResultInput struct {
	TaskId uint64 `path:"task_id" json:"task_id" description:"Task id" validate:"required"`
}

type ResultInputWithSignature struct {
	ResultInput
	Timestamp int64  `form:"timestamp" json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `form:"signature" json:"signature" description:"Signature" validate:"required"`
}

func UploadResult(ctx *gin.Context, in *ResultInputWithSignature) (*response.Response, error) {

	match, address, err := ValidateSignature(in.ResultInput, in.Timestamp, in.Signature)

	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	if !match {
		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	var task models.InferenceTask

	if result := config.GetDB().Where(&models.InferenceTask{TaskId: in.TaskId}).First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			validationErr := response.NewValidationErrorResponse("task_id", "Task not found")
			return nil, validationErr
		} else {
			return nil, response.NewExceptionResponse(result.Error)
		}
	}

	if task.Status != models.InferenceTaskPendingResults {
		validationErr := response.NewValidationErrorResponse("task_id", "Task not success")
		return nil, validationErr
	}

	resultNode := &models.SelectedNode{
		InferenceTaskID:  task.ID,
		IsResultSelected: true,
	}

	if err := config.GetDB().Where(resultNode).First(resultNode).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewValidationErrorResponse("task_id", "Task not ready")
		} else {
			return nil, response.NewExceptionResponse(err)
		}
	}

	if resultNode.NodeAddress != address {
		validationErr := response.NewValidationErrorResponse("signature", "Signer not allowed")
		return nil, validationErr
	}

	form, _ := ctx.MultipartForm()
	files := form.File["images"]

	// Check whether the images are correct
	var resultHashBytes []byte

	for _, file := range files {

		fileObj, err := file.Open()

		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		var hash []byte
		if task.TaskType == models.TaskTypeSD {
			hash, err = blockchain.GetPHashForImage(fileObj)
		} else {
			hash, err = blockchain.GetHashForGPTResponse(fileObj)
		}

		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		resultHashBytes = append(resultHashBytes, hash...)

		err = fileObj.Close()
		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}
	}

	uploadedResult := hexutil.Encode(resultHashBytes)

	log.Debugln("image compare: result from the blockchain: " + resultNode.Result)
	log.Debugln("image compare: result from the uploaded file: " + uploadedResult)

	if resultNode.Result != uploadedResult {
		validationErr := response.NewValidationErrorResponse("images", "Wrong images uploaded")
		return nil, validationErr
	}

	appConfig := config.GetConfig()

	taskWorkspace := appConfig.DataDir.InferenceTasks
	taskIdStr := task.GetTaskIdAsString()

	taskDir := filepath.Join(taskWorkspace, taskIdStr, "results")
	if err = os.MkdirAll(taskDir, os.ModePerm); err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	fileNum := 0
	var fileExt string
	if task.TaskType == models.TaskTypeSD || task.TaskType == models.TaskTypeSDFT{
		fileExt = ".png"
	} else {
		fileExt = ".json"
	}

	for _, file := range files {
		filename := filepath.Join(taskDir, strconv.Itoa(fileNum)+fileExt)
		if err := ctx.SaveUploadedFile(file, filename); err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		fileNum += 1
	}

	// store checkpoint of finetune type task
	if task.TaskType == models.TaskTypeSDFT {
		checkpointFiles, ok := form.File["checkpoint"]
		if !ok {
			return nil, response.NewValidationErrorResponse("checkpoint", "Checkpoint not uploaded")
		}
		if len(checkpointFiles) != 1 {
			return nil, response.NewValidationErrorResponse("checkpoint", "Too many checkpoint files")
		}
		checkpointFile := checkpointFiles[0]
		checkpointFilename := filepath.Join(taskDir, "checkpoint.zip")
		if err := ctx.SaveUploadedFile(checkpointFile, checkpointFilename); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
	}

	// Update task status
	task.Status = models.InferenceTaskResultsUploaded

	if err := config.GetDB().Save(&task).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &response.Response{}, nil
}
