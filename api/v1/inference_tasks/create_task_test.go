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
	assert.Equal(t, err, nil)

	task, err := v1.PrepareTask(addresses)
	assert.Equal(t, err, nil)

	taskInput := inference_tasks.TaskInput{TaskId: task.TaskId, SelectedNodes: task.SelectedNodes, TaskParams: task.TaskParams}

	signBytes, err := json.Marshal(taskInput)
	assert.Equal(t, err, nil)

	timestamp, signature, err := v1.SignData(signBytes, privateKeys[0])
	assert.Equal(t, err, nil)

	r := callApi(
		task.TaskId,
		"",
		task.SelectedNodes,
		task.Creator,
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "task_params", "task_params is required")
}

func callApi(taskId int64, taskParams string, selectedNodes string, signer string, timestamp int64, signature string) *httptest.ResponseRecorder {

	data := url.Values{}
	data.Set("taskId", strconv.FormatInt(taskId, 10))
	data.Set("taskParams", taskParams)
	data.Set("selectedNodes", selectedNodes)
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
