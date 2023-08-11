package inference_tasks_test

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, nil, err, "prepare accounts error")

	task, err := v1.PrepareTask(addresses)
	assert.Equal(t, nil, err, "prepare task error")

	taskInput := inference_tasks.TaskInput{TaskId: task.TaskId, SelectedNodes: task.SelectedNodes, TaskParams: task.TaskParams}

	signBytes, err := json.Marshal(taskInput)
	assert.Equal(t, nil, err, "task input json marshall error")

	// Missing argument

	timestamp, signature, err := v1.SignData(signBytes, privateKeys[0])
	assert.Equal(t, nil, err, "sign data error")

	r := callCreateTaskApi(
		task.TaskId,
		"",
		task.SelectedNodes,
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "task_params", "required")

	// Late timestamp

	timestamp = timestamp - 100

	r = callCreateTaskApi(
		task.TaskId,
		task.TaskParams,
		task.SelectedNodes,
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "signature", "Invalid signature")

	// Successful creation

	log.Debugln("signing using address: " + addresses[0])
	timestamp, signature, err = v1.SignData(signBytes, privateKeys[0])
	assert.Equal(t, nil, err, "sign data error")

	r = callCreateTaskApi(
		task.TaskId,
		task.TaskParams,
		task.SelectedNodes,
		timestamp,
		signature)

	v1.AssertTaskResponse(t, r, task)

	t.Cleanup(tests.ClearDB)
}

func callCreateTaskApi(taskId int64, taskParams string, selectedNodes string, timestamp int64, signature string) *httptest.ResponseRecorder {

	data := url.Values{}
	data.Set("task_id", strconv.FormatInt(taskId, 10))
	data.Set("task_params", taskParams)
	data.Set("selected_nodes", selectedNodes)
	data.Set("timestamp", strconv.FormatInt(timestamp, 10))
	data.Set("signature", signature)

	req, _ := http.NewRequest("POST", "/v1/inference_tasks", strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}
