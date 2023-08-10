package inference_tasks_test

import (
	"encoding/json"
	"github.com/magiconair/properties/assert"
	"h_relay/api/v1/inference_tasks"
	"h_relay/tests"
	v1 "h_relay/tests/api/v1"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestCreateTaskResponse(t *testing.T) {

	addresses, privateKeys, err := v1.PrepareAccounts()
	assert.Equal(t, err, nil, "prepare accounts error")

	task, err := v1.PrepareTask(addresses)
	assert.Equal(t, err, nil, "prepare task error")

	taskInput := inference_tasks.TaskInput{TaskId: task.TaskId, SelectedNodes: task.SelectedNodes, TaskParams: task.TaskParams}

	signBytes, err := json.Marshal(taskInput)
	assert.Equal(t, err, nil, "task input json marshall error")

	timestamp, signature, err := v1.SignData(signBytes, privateKeys[0])
	assert.Equal(t, err, nil, "sign data error")

	r := callApi(
		task.TaskId,
		"",
		task.SelectedNodes,
		task.Creator,
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "task_params", "required")

	timestamp = timestamp - 100

	r = callApi(
		task.TaskId,
		task.TaskParams,
		task.SelectedNodes,
		task.Creator,
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "signature", "Invalid signature")

	timestamp = timestamp + 100

	r = callApi(
		task.TaskId,
		task.TaskParams,
		task.SelectedNodes,
		task.Creator,
		timestamp,
		signature+"e")

	v1.AssertValidationErrorResponse(t, r, "signature", "Invalid signature")

	r = callApi(
		task.TaskId,
		task.TaskParams,
		task.SelectedNodes,
		task.Creator,
		timestamp,
		signature)

	taskResponse := &inference_tasks.TaskResponse{}

	err = json.Unmarshal(r.Body.Bytes(), taskResponse)
	assert.Equal(t, err, nil)

	assert.Equal(t, taskResponse.GetMessage(), "success")
	assert.Equal(t, taskResponse.Data.TaskId, task.TaskId)
	assert.Equal(t, taskResponse.Data.TaskParams, task.TaskParams)
	assert.Equal(t, taskResponse.Data.CreatedAt.IsZero(), false)
	assert.Equal(t, taskResponse.Data.UpdatedAt.IsZero(), false)
}

func callApi(taskId int64, taskParams string, selectedNodes string, signer string, timestamp int64, signature string) *httptest.ResponseRecorder {

	data := url.Values{}
	data.Set("task_id", strconv.FormatInt(taskId, 10))
	data.Set("task_params", taskParams)
	data.Set("selected_nodes", selectedNodes)
	data.Set("signer", signer)
	data.Set("timestamp", strconv.FormatInt(timestamp, 10))
	data.Set("signature", signature)

	req, _ := http.NewRequest("POST", "/v1/inference_tasks", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}
