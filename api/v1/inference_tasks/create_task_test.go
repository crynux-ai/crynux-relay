package inference_tasks_test

import (
	"bytes"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"h_relay/api/v1/inference_tasks"
	"h_relay/config"
	"h_relay/tests"
	v1 "h_relay/tests/api/v1"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTaskBeforeBlockchainConfirmation(t *testing.T) {

	task := tests.PrepareRandomTask()

	_, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare account error")

	timestamp, signature, err := v1.SignData(task, privateKeys[0])

	r := callCreateTaskApi(t, task, timestamp, signature)
	v1.AssertValidationErrorResponse(t, r, "task_id", "Task not found on the Blockchain")

	t.Cleanup(tests.ClearDB)
}

func TestCreateTaskAfterBlockchainConfirmation(t *testing.T) {

	addresses, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare account error")

	taskInput, task, err := tests.PrepareBlockchainConfirmedTask(addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	timestamp, signature, err := v1.SignData(taskInput, privateKeys[0])
	r := callCreateTaskApi(t, taskInput, timestamp, signature)

	task.BaseModel = taskInput.BaseModel
	task.LoraModel = taskInput.LoraModel
	task.Prompt = taskInput.Prompt
	task.TaskConfig = &taskInput.TaskConfig
	task.Pose = &taskInput.Pose

	v1.AssertTaskResponse(t, r, task)

	t.Cleanup(tests.ClearDB)
}

func TestCreateTaskUsingUnauthorizedAccount(t *testing.T) {
	addresses, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare account error")

	taskInput, _, err := tests.PrepareBlockchainConfirmedTask(addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	timestamp, signature, err := v1.SignData(taskInput, privateKeys[1])
	r := callCreateTaskApi(t, taskInput, timestamp, signature)

	v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")

	t.Cleanup(tests.ClearDB)
}

func TestCreateDuplicateTask(t *testing.T) {

	addresses, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare account error")

	taskInput, task, err := tests.PrepareBlockchainConfirmedTask(addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	timestamp, signature, err := v1.SignData(taskInput, privateKeys[0])
	r := callCreateTaskApi(t, taskInput, timestamp, signature)

	task.BaseModel = taskInput.BaseModel
	task.LoraModel = taskInput.LoraModel
	task.Prompt = taskInput.Prompt
	task.TaskConfig = &taskInput.TaskConfig
	task.Pose = &taskInput.Pose

	v1.AssertTaskResponse(t, r, task)

	timestamp, signature, err = v1.SignData(taskInput, privateKeys[0])
	r = callCreateTaskApi(t, taskInput, timestamp, signature)

	v1.AssertValidationErrorResponse(t, r, "task_id", "Task already uploaded")

	t.Cleanup(tests.ClearDB)
}

func TestCreateTaskWithMismatchedParamHash(t *testing.T) {
	addresses, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare account error")

	taskInput, _, err := tests.PrepareBlockchainConfirmedTask(addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	oldPrompt := taskInput.Prompt
	taskInput.Prompt += ", in anime style"

	timestamp, signature, err := v1.SignData(taskInput, privateKeys[0])
	assert.Equal(t, nil, err, "sign data error")

	r := callCreateTaskApi(t, taskInput, timestamp, signature)

	v1.AssertValidationErrorResponse(t, r, "data_hash", "Data hash mismatch")

	taskInput.Prompt = oldPrompt
	taskInput.TaskConfig.Steps = 60

	timestamp, signature, err = v1.SignData(taskInput, privateKeys[0])
	assert.Equal(t, nil, err, "sign data error")

	r = callCreateTaskApi(t, taskInput, timestamp, signature)
	v1.AssertValidationErrorResponse(t, r, "task_hash", "Task hash mismatch")

	t.Cleanup(tests.ClearDB)
}

func callCreateTaskApi(t *testing.T, task *inference_tasks.TaskInput, timestamp int64, signature string) *httptest.ResponseRecorder {

	inputWithSignature := &inference_tasks.TaskInputWithSignature{
		TaskInput: *task,
		Timestamp: timestamp,
		Signature: signature,
	}

	jsonBytes, err := json.Marshal(inputWithSignature)
	assert.Equal(t, nil, err, "json marshall error")

	log.Debugln("post data: " + string(jsonBytes))

	req, _ := http.NewRequest("POST", "/v1/inference_tasks", bytes.NewReader(jsonBytes))
	req.Header.Add("Content-Type", "application/json")

	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}
