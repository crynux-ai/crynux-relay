package inference_tasks_test

import (
	"encoding/json"
	"github.com/magiconair/properties/assert"
	"h_relay/api/v1/inference_tasks"
	"h_relay/config"
	"h_relay/tests"
	v1 "h_relay/tests/api/v1"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestGetTaskById(t *testing.T) {

	addresses, privateKeys, err := v1.PrepareAccounts()
	assert.Equal(t, err, nil, "prepare accounts error")

	task, err := v1.PrepareTask(addresses)
	assert.Equal(t, err, nil, "prepare task error")

	err = config.GetDB().Create(task).Error
	assert.Equal(t, err, nil, "save task to db error")

	taskInput := inference_tasks.GetTaskInput{TaskId: 567}

	signBytes, err := json.Marshal(taskInput)
	assert.Equal(t, err, nil, "task input json marshall error")

	// Get a non-exist task

	timestamp, signature, err := v1.SignData(signBytes, privateKeys[0])
	assert.Equal(t, err, nil, "sign data error")

	r := callGetTaskByIdApi(
		567,
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "task_id", "Task not found")

	// Get the task using an unauthorized account

	taskInput.TaskId = task.TaskId
	signBytes, err = json.Marshal(taskInput)
	assert.Equal(t, err, nil, "task input json marshall error")

	timestamp, signature, err = v1.SignData(signBytes, privateKeys[4])
	assert.Equal(t, err, nil, "sign data error")

	r = callGetTaskByIdApi(
		task.TaskId,
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")

	// Get the task using the creator's account

	timestamp, signature, err = v1.SignData(signBytes, privateKeys[0])
	assert.Equal(t, err, nil, "sign data error")

	r = callGetTaskByIdApi(
		task.TaskId,
		timestamp,
		signature)

	v1.AssertTaskResponse(t, r, task)

	// Get the task using the selected node's account
	timestamp, signature, err = v1.SignData(signBytes, privateKeys[2])
	assert.Equal(t, err, nil, "sign data error")

	r = callGetTaskByIdApi(
		task.TaskId,
		timestamp,
		signature)

	v1.AssertTaskResponse(t, r, task)

	t.Cleanup(tests.ClearDB)
}

func callGetTaskByIdApi(taskId int64, timestamp int64, signature string) *httptest.ResponseRecorder {

	taskIdStr := strconv.FormatInt(taskId, 10)

	data := url.Values{}

	data.Set("timestamp", strconv.FormatInt(timestamp, 10))
	data.Set("signature", signature)

	req, _ := http.NewRequest("GET", "/v1/inference_tasks/"+taskIdStr+"?"+data.Encode(), nil)

	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}
