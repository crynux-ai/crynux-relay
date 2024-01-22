package inference_tasks_test

import (
	"bytes"
	"crynux_relay/api/v1/inference_tasks"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/tests"
	v1 "crynux_relay/tests/api/v1"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCreateTaskBeforeBlockchainConfirmation(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		task, err := tests.PrepareRandomTask(taskType)
		assert.Equal(t, nil, err, "prepare task error")

		_, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "prepare account error")

		timestamp, signature, err := v1.SignData(task, privateKeys[0])

		r := callCreateTaskApi(t, task, timestamp, signature)
		v1.AssertValidationErrorResponse(t, r, "task_id", "Task not found on the Blockchain")

		t.Cleanup(tests.ClearDB)
	}
}

func TestCreateTaskAfterBlockchainConfirmation(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "prepare account error")

		taskInput, task, err := tests.PrepareBlockchainConfirmedTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "prepare task error")

		timestamp, signature, err := v1.SignData(taskInput, privateKeys[0])
		r := callCreateTaskApi(t, taskInput, timestamp, signature)

		task.TaskArgs = taskInput.TaskArgs

		v1.AssertTaskResponse(t, r, task)

		t.Cleanup(tests.ClearDB)
	}
}

func TestCreateTaskUsingUnauthorizedAccount(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "prepare account error")

		taskInput, _, err := tests.PrepareBlockchainConfirmedTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "prepare task error")

		timestamp, signature, err := v1.SignData(taskInput, privateKeys[1])
		r := callCreateTaskApi(t, taskInput, timestamp, signature)

		v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")

		t.Cleanup(tests.ClearDB)
	}
}

func TestCreateDuplicateTask(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "prepare account error")

		taskInput, task, err := tests.PrepareBlockchainConfirmedTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "prepare task error")

		timestamp, signature, err := v1.SignData(taskInput, privateKeys[0])
		r := callCreateTaskApi(t, taskInput, timestamp, signature)

		task.TaskArgs = taskInput.TaskArgs

		v1.AssertTaskResponse(t, r, task)

		timestamp, signature, err = v1.SignData(taskInput, privateKeys[0])
		r = callCreateTaskApi(t, taskInput, timestamp, signature)

		v1.AssertValidationErrorResponse(t, r, "task_id", "Task already uploaded")

		t.Cleanup(tests.ClearDB)
	}
}

func TestCreateTaskWithMismatchedParamHash(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "prepare account error")

		taskInput, _, err := tests.PrepareBlockchainConfirmedTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "prepare task error")

		var taskArgsMap map[string]interface{}
		err = json.Unmarshal([]byte(taskInput.TaskArgs), &taskArgsMap)
		assert.Equal(t, nil, err, "unmarshall task json error")

		if taskType == models.TaskTypeSD {
			oldPrompt := taskArgsMap["prompt"].(string)
			taskArgsMap["prompt"] = oldPrompt + ", in anime style"
			newTaskArgsJson, err := json.Marshal(taskArgsMap)
			assert.Equal(t, nil, err, "marshall new task json error")
			taskInput.TaskArgs = string(newTaskArgsJson)

			timestamp, signature, err := v1.SignData(taskInput, privateKeys[0])
			assert.Equal(t, nil, err, "sign data error")

			r := callCreateTaskApi(t, taskInput, timestamp, signature)

			v1.AssertValidationErrorResponse(t, r, "task_hash", "Task hash mismatch")

			taskArgsMap["prompt"] = oldPrompt
			taskArgsMap["task_config"].(map[string]interface{})["steps"] = 60

			newTaskArgsJson, err = json.Marshal(taskArgsMap)
			assert.Equal(t, nil, err, "marshall new task json error")
			taskInput.TaskArgs = string(newTaskArgsJson)

			timestamp, signature, err = v1.SignData(taskInput, privateKeys[0])
			assert.Equal(t, nil, err, "sign data error")

			r = callCreateTaskApi(t, taskInput, timestamp, signature)
			v1.AssertValidationErrorResponse(t, r, "task_hash", "Task hash mismatch")
		} else {
			oldModel := taskArgsMap["model"].(string)
			taskArgsMap["model"] = "gpt3"

			newTaskArgsJson, err := json.Marshal(taskArgsMap)
			assert.Equal(t, nil, err, "marshall new task json error")
			taskInput.TaskArgs = string(newTaskArgsJson)

			timestamp, signature, err := v1.SignData(taskInput, privateKeys[0])
			assert.Equal(t, nil, err, "sign data error")

			r := callCreateTaskApi(t, taskInput, timestamp, signature)

			v1.AssertValidationErrorResponse(t, r, "task_hash", "Task hash mismatch")

			taskArgsMap["model"] = oldModel
			taskArgsMap["seed"] = 0

			newTaskArgsJson, err = json.Marshal(taskArgsMap)
			assert.Equal(t, nil, err, "marshall new task json error")
			taskInput.TaskArgs = string(newTaskArgsJson)

			timestamp, signature, err = v1.SignData(taskInput, privateKeys[0])
			assert.Equal(t, nil, err, "sign data error")

			r = callCreateTaskApi(t, taskInput, timestamp, signature)
			v1.AssertValidationErrorResponse(t, r, "task_hash", "Task hash mismatch")
		}

		t.Cleanup(tests.ClearDB)
	}
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
