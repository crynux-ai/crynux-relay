package inference_tasks

import (
	"crynux_relay/api/v1/response"
	"crynux_relay/api/v1/validate"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/service"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type ValidateTaskInput struct {
	TaskIDCommitments []string `json:"task_id_commitments" description:"task_id_commitments" validate:"required"`
	TaskID            string   `json:"task_id" description:"task_id" validate:"required"`
	VrfProof          string   `json:"vrf_proof" description:"vrf_proof" validate:"required"`
	PublicKey         string   `json:"public_key" description:"public_key" validate:"required"`
}

type ValidateTaskInputWithSignature struct {
	ValidateTaskInput
	Timestamp int64  `json:"timestamp" description:"Signature timestamp" validate:"required"`
	Signature string `json:"signature" description:"Signature" validate:"required"`
}

func ValidateTask(c *gin.Context, in *ValidateTaskInputWithSignature) (*response.Response, error) {
	if len(in.TaskIDCommitments) != 1 && len(in.TaskIDCommitments) != 3 {
		return nil, response.NewValidationErrorResponse("task_id_commitments", "TaskIDCommitments length incorrect")
	}

	match, address, err := validate.ValidateSignature(in.ValidateTaskInput, in.Timestamp, in.Signature)

	if err != nil || !match {

		if err != nil {
			log.Debugln("error in sig validate: " + err.Error())
		}

		validationErr := response.NewValidationErrorResponse("signature", "Invalid signature")
		return nil, validationErr
	}

	var tasks []*models.InferenceTask
	for _, taskIDCommitment := range in.TaskIDCommitments {
		task, err := models.GetTaskByIDCommitment(c.Request.Context(), config.GetDB(), taskIDCommitment)
		if err != nil {
			return nil, response.NewExceptionResponse(err)
		}
		if len(task.TaskArgs) == 0 {
			return nil, response.NewValidationErrorResponse("task_id_commitment", "Task not ready")
		}
		if task.Creator != address {
			return nil, response.NewValidationErrorResponse("signature", "Signer not allowed")
		}
		tasks = append(tasks, task)
	}

	if len(tasks) == 1 {
		if err := service.ValidateSingleTask(c.Request.Context(), tasks[0], in.TaskID, in.VrfProof, in.PublicKey); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
	} else if len(tasks) == 3 {
		if err := service.ValidateTaskGroup(c.Request.Context(), tasks, in.TaskID, in.VrfProof, in.PublicKey); err != nil {
			return nil, response.NewExceptionResponse(err)
		}
	}
	return &response.Response{}, nil
}
