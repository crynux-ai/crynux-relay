package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type TaskInput struct {
	TaskArgs         string `json:"task_args" description:"Task arguments" validate:"required"`
	TaskIDCommitment string `json:"task_id_commitment" description:"Task id commitment" validate:"required"`
}

type TaskInputWithSignature struct {
	TaskInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func CreateTask(ctx *gin.Context, in *TaskInputWithSignature) (*TaskResponse, error) {

	match, address, err := ValidateSignature(in.TaskInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	bs, err := hexutil.Decode(in.TaskIDCommitment)
	if err != nil {
		return nil, response.NewValidationErrorResponse("taskIDCommitment", "Invalid task id commitment")
	}
	if len(bs) != 32 {
		return nil, response.NewValidationErrorResponse("taskIDCommitment", "Invalid task id commitment")
	}
	taskIDCommitmentBytes := (*[32]byte)(bs)

	taskInstance, err := blockchain.GetTaskContractInstance()
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	chainTask, err := taskInstance.GetTask(&bind.CallOpts{
		Pending: false,
		Context: ctx.Request.Context(),
	}, *taskIDCommitmentBytes)
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if hexutil.Encode(chainTask.TaskIDCommitment[:]) != in.TaskIDCommitment {
		return nil,
			response.NewValidationErrorResponse(
				"task_id_commitment",
				"Task not found on the Blockchain")
	}
	if address != chainTask.Creator.Hex() {
		return nil, response.NewValidationErrorResponse("signature", "Signer not allowed")
	}

	validationErr, err := models.ValidateTaskArgsJsonStr(in.TaskArgs, models.ChainTaskType(chainTask.TaskType))
	if err != nil {
		return nil, response.NewExceptionResponse(err)
	}
	if validationErr != nil {
		return nil, response.NewValidationErrorResponse("task_args", validationErr.Error())
	}

	task := models.InferenceTask{}

	err = config.GetDB().Model(&models.InferenceTask{}).Where("task_id_commitment = ?", in.TaskIDCommitment).First(&task).Error
	if err == nil {
		return nil, response.NewValidationErrorResponse("task_id_commitment", "Task already uploaded")
	}
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewExceptionResponse(err)
		}
	}

	taskFee, _ := utils.WeiToEther(chainTask.TaskFee).Float64()

	task.TaskArgs = in.TaskArgs
	task.TaskIDCommitment = in.TaskIDCommitment
	task.Creator = chainTask.Creator.Hex()
	task.Status = models.InferenceTaskCreated
	task.TaskType = models.ChainTaskType(chainTask.TaskType)
	task.MinVRAM = chainTask.MinimumVRAM.Uint64()
	task.RequiredGPU = chainTask.RequiredGPU
	task.RequiredGPUVRAM = chainTask.RequiredGPUVRAM.Uint64()
	task.TaskFee = taskFee
	task.TaskSize = chainTask.TaskSize.Uint64()

	if err := config.GetDB().Save(&task).Error; err != nil {
		return nil, response.NewExceptionResponse(err)
	}

	return &TaskResponse{Data: task}, nil
}
