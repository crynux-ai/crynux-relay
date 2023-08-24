package inference_tasks

import (
	"errors"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/api/v1/response"
	"h_relay/blockchain"
	"h_relay/config"
	"h_relay/models"
	"os"
	"path/filepath"
	"strconv"
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
	var pHashBytes []byte

	for _, file := range files {

		imageFile, err := file.Open()

		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		pHash, err := blockchain.GetPHashForImage(imageFile)

		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		pHashBytes = append(pHashBytes, pHash...)

		err = imageFile.Close()
		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}
	}

	uploadedResult := hexutil.Encode(pHashBytes)

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
	if err = os.MkdirAll(taskDir, os.ModeDir); err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	fileNum := 0

	for _, file := range files {
		filename := filepath.Join(taskDir, strconv.Itoa(fileNum)+".png")
		if err := ctx.SaveUploadedFile(file, filename); err != nil {
			return nil, response.NewExceptionResponse(err)
		}

		fileNum += 1
	}

	return &response.Response{}, nil
}
