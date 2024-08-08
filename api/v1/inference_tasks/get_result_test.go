package inference_tasks_test

import (
	"crynux_relay/api/v1/inference_tasks"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/tests"
	v1 "crynux_relay/tests/api/v1"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnauthorizedGetImage(t *testing.T) {
	addresses, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare accounts error")

	_, task, err := tests.PrepareResultUploadedTask(models.TaskTypeSD, addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	getResultInput := &inference_tasks.GetSDResultInput{
		TaskId:   task.TaskId,
		ImageNum: "0",
	}

	timestamp, signature, err := v1.SignData(getResultInput, privateKeys[1])
	assert.Equal(t, nil, err, "sign data error")

	r := callGetImageApi(
		task.GetTaskIdAsString(),
		"0",
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")

	t.Cleanup(func() {
		tests.ClearDB()
		if err := tests.ClearDataFolders(); err != nil {
			t.Error(err)
		}
	})
}

func TestUnauthorizedGetGPTResponse(t *testing.T) {
	addresses, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare accounts error")

	_, task, err := tests.PrepareResultUploadedTask(models.TaskTypeLLM, addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	getResultInput := &inference_tasks.GetGPTResultInput{
		TaskId: task.TaskId,
	}

	timestamp, signature, err := v1.SignData(getResultInput, privateKeys[1])
	assert.Equal(t, nil, err, "sign data error")

	r := callGetGPTResponseApi(
		task.GetTaskIdAsString(),
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")

	t.Cleanup(func() {
		tests.ClearDB()
		if err := tests.ClearDataFolders(); err != nil {
			t.Error(err)
		}
	})
}

func TestGetImage(t *testing.T) {	
	t.Cleanup(func() {
		tests.ClearDB()
		if err := tests.ClearDataFolders(); err != nil {
			t.Error(err)
		}
	})

	addresses, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare accounts error")

	_, task, err := tests.PrepareResultUploadedTask(models.TaskTypeSD, addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	imageNum := "2"
	srcFile := "2.png"
	dstFile := "downloaded.png"

	getResultInput := &inference_tasks.GetSDResultInput{
		TaskId:   task.TaskId,
		ImageNum: imageNum,
	}

	timestamp, signature, err := v1.SignData(getResultInput, privateKeys[0])
	assert.Equal(t, nil, err, "sign data error")

	r := callGetImageApi(
		task.GetTaskIdAsString(),
		imageNum,
		timestamp,
		signature)

	assert.Equal(t, 200, r.Code, "wrong http status code. message: "+r.Body.String())

	appConfig := config.GetConfig()
	imageFolder := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.GetTaskIdAsString(),
		"results",
	)

	out, err := os.Create(filepath.Join(imageFolder, dstFile))
	assert.Equal(t, nil, err, "create tmp file error")

	_, err = io.Copy(out, r.Body)
	assert.Equal(t, nil, err, "write tmp file error")

	err = out.Close()
	assert.Equal(t, nil, err, "close tmp file error")

	originalFile, err := os.Stat(filepath.Join(imageFolder, srcFile))
	assert.Equal(t, nil, err, "read original file error")

	downloadedFile, err := os.Stat(filepath.Join(imageFolder, dstFile))
	assert.Equal(t, nil, err, "read downloaded file error")

	assert.Equal(t, originalFile.Size(), downloadedFile.Size(), "different file sizes")
}

func TestGetGPTResponse(t *testing.T) {
	t.Cleanup(func() {
		tests.ClearDB()
		if err := tests.ClearDataFolders(); err != nil {
			t.Error(err)
		}
	})

	addresses, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare accounts error")

	_, task, err := tests.PrepareResultUploadedTask(models.TaskTypeLLM, addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	getResultInput := &inference_tasks.GetGPTResultInput{
		TaskId: task.TaskId,
	}

	timestamp, signature, err := v1.SignData(getResultInput, privateKeys[0])
	assert.Equal(t, nil, err, "sign data error")

	r := callGetGPTResponseApi(
		task.GetTaskIdAsString(),
		timestamp,
		signature)

	assert.Equal(t, 200, r.Code, "wrong http status code. message: "+r.Body.String())

	res := inference_tasks.GPTResultResponse{}
	if err := json.Unmarshal(r.Body.Bytes(), &res); err != nil {
		t.Error(err)
	}
	target := models.GPTTaskResponse{}
	if err := json.Unmarshal([]byte(tests.GPTResponseStr), &target); err != nil {
		t.Error(err)
	}
	assert.Equal(t, target, res.Data, "wrong returned gpt response")
}

func callGetImageApi(
	taskIdStr string,
	imageNum string,
	timestamp int64,
	signature string) *httptest.ResponseRecorder {

	endpoint := "/v1/inference_tasks/stable_diffusion/" + taskIdStr + "/results/" + imageNum
	query := "?timestamp=" + strconv.FormatInt(timestamp, 10) + "&signature=" + signature

	req, _ := http.NewRequest("GET", endpoint+query, nil)
	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}

func callGetGPTResponseApi(taskIdStr string, timestamp int64, signature string) *httptest.ResponseRecorder {
	endpoint := "/v1/inference_tasks/gpt/" + taskIdStr + "/results"
	query := "?timestamp=" + strconv.FormatInt(timestamp, 10) + "&signature=" + signature

	req, _ := http.NewRequest("GET", endpoint+query, nil)
	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}
