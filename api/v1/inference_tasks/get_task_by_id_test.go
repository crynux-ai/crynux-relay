package inference_tasks_test

import (
	"crynux_relay/api/v1/inference_tasks"
	"crynux_relay/config"
	"crynux_relay/tests"
	v1 "crynux_relay/tests/api/v1"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBlockchainConfirmedTask(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "error preparing accounts")

		taskInput, _, err := tests.PrepareBlockchainConfirmedTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "error preparing task")

		getResultInput := inference_tasks.GetTaskInput{TaskId: taskInput.TaskId}

		timestamp, signature, err := v1.SignData(getResultInput, privateKeys[1])

		r := callGetTaskByIdApi(taskInput.TaskId, timestamp, signature)
		v1.AssertValidationErrorResponse(t, r, "task_id", "Task not ready")

		t.Cleanup(tests.ClearDB)
	}
}

func TestGetParamsUploadedTask(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "error preparing accounts")

		taskInput, task, err := tests.PrepareParamsUploadedTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "error preparing task")

		getResultInput := inference_tasks.GetTaskInput{TaskId: taskInput.TaskId}

		timestamp, signature, err := v1.SignData(getResultInput, privateKeys[1])

		r := callGetTaskByIdApi(taskInput.TaskId, timestamp, signature)
		v1.AssertTaskResponse(t, r, task)

		t.Cleanup(tests.ClearDB)
	}
}

func TestGetUnauthorizedTask(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "error preparing accounts")

		taskInput, _, err := tests.PrepareParamsUploadedTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "error preparing task")

		getResultInput := inference_tasks.GetTaskInput{TaskId: taskInput.TaskId}

		timestamp, signature, err := v1.SignData(getResultInput, privateKeys[4])

		r := callGetTaskByIdApi(taskInput.TaskId, timestamp, signature)
		v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")

		t.Cleanup(tests.ClearDB)
	}
}

func callGetTaskByIdApi(taskId uint64, timestamp int64, signature string) *httptest.ResponseRecorder {

	taskIdStr := strconv.FormatUint(taskId, 10)

	data := url.Values{}

	data.Set("timestamp", strconv.FormatInt(timestamp, 10))
	data.Set("signature", signature)

	req, _ := http.NewRequest("GET", "/v1/inference_tasks/"+taskIdStr+"?"+data.Encode(), nil)

	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}
