package inference_tasks

import (
	"context"
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/validate"
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"database/sql"
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type TaskInput struct {
	TaskIDCommitment string          `path:"task_id_commitment" json:"task_id_commitment" description:"Task id commitment" validate:"required"`
	TaskArgs         string          `json:"task_args" description:"Task arguments" validate:"required"`
	TaskType         models.TaskType `json:"task_type" description:"Task type" validate:"required"`
	Nonce            string          `json:"nonce" description:"nonce" validate:"required"`
	TaskModelIDs     []string        `json:"task_model_ids" description:"task model ids" validate:"required"`
	MinVram          uint64          `json:"min_vram" description:"min vram" validate:"required"`
	RequiredGPU      bool            `json:"required_gpu" description:"required gpu" validate:"required"`
	RequiredGPUVram  uint64          `json:"required_gpu_vram" description:"required gpu vram" validate:"required"`
	TaskVersion      string          `json:"task_version" description:"task version" validate:"required"`
	TaskSize         uint64          `json:"task_size" description:"task size" validate:"required"`
}

type TaskInputWithSignature struct {
	TaskInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func CreateTask(c *gin.Context, in *TaskInputWithSignature) (*TaskResponse, error) {

	match, address, err := validate.ValidateSignature(in.TaskInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	taskIDCommitmentBytes, err := utils.HexStrToCommitment(in.TaskIDCommitment)
	if err != nil {
		return nil, response.NewValidationErrorResponse("task_id_commitment", "Invalid task id commitment")
	}

	chainCtx, chainCancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer chainCancel()
	chainTask, err := blockchain.GetTaskByCommitment(chainCtx, *taskIDCommitmentBytes)
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if hexutil.Encode(chainTask.TaskIDCommitment[:]) != in.TaskIDCommitment {
		return nil,
			response.NewValidationErrorResponse(
				"task_id_commitment",
				"Task not found on the Blockchain")
	}
	if models.TaskStatus(chainTask.Status) != models.TaskStarted {
		return nil, response.NewValidationErrorResponse("task_id_commitment", "Task not started")
	}

	if address != chainTask.Creator.Hex() {
		return nil, response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	validationErr, err := models.ValidateTaskArgsJsonStr(in.TaskArgs, models.TaskType(chainTask.TaskType))
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if validationErr != nil {
		return nil, response.NewValidationErrorResponse("task_args", validationErr.Error())
	}

	_, err = models.GetTaskByIDCommitment(c.Request.Context(), config.GetDB(), in.TaskIDCommitment)
	if err == nil {
		return nil, response.NewValidationErrorResponse("task_id_commitment", "Task already uploaded")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.NewExceptionResponse(err)
	}

	if models.TaskType(chainTask.TaskType) == models.TaskTypeSDFTLora {
		form, err := c.MultipartForm()
		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		var checkpoint *multipart.FileHeader
		if files, ok := form.File["checkpoint"]; !ok {
			return nil, response.NewValidationErrorResponse("checkpoint", "Checkpoint not uploaded")
		} else {
			if len(files) != 1 {
				return nil, response.NewValidationErrorResponse("checkpoint", "More than one checkpoint file")
			}
			checkpoint = files[0]
		}

		appConfig := config.GetConfig()

		taskDir := filepath.Join(appConfig.DataDir.InferenceTasks, in.TaskIDCommitment, "input")
		if err = os.MkdirAll(taskDir, 0o711); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		checkpointFilename := filepath.Join(taskDir, "checkpoint.zip")
		if err := c.SaveUploadedFile(checkpoint, checkpointFilename); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
	}

	taskFee, _ := utils.WeiToEther(chainTask.TaskFee).Float64()

	task := &models.InferenceTask{}
	task.TaskArgs = in.TaskArgs
	task.TaskIDCommitment = in.TaskIDCommitment
	task.Creator = chainTask.Creator.Hex()
	task.Status = models.TaskQueued
	task.TaskType = models.TaskType(chainTask.TaskType)
	task.MinVRAM = chainTask.MinimumVRAM.Uint64()
	task.RequiredGPU = chainTask.RequiredGPU
	task.RequiredGPUVRAM = chainTask.RequiredGPUVRAM.Uint64()
	task.TaskFee = taskFee
	task.TaskSize = chainTask.TaskSize.Uint64()
	task.ModelIDs = models.StringArray(chainTask.ModelIDs)
	task.SelectedNode = chainTask.SelectedNode.Hex()
	task.CreateTime = sql.NullTime{
		Time:  time.Unix(chainTask.CreateTimestamp.Int64(), 0),
		Valid: true,
	}

	if err := task.Save(c.Request.Context(), config.GetDB()); err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &TaskResponse{Data: *task}, nil
}
