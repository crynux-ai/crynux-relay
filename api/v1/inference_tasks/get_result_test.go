package inference_tasks_test

import (
	"crynux_relay/api/v1/inference_tasks"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/tests"
	v1 "crynux_relay/tests/api/v1"
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
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "prepare accounts error")

		_, task, err := tests.PrepareResultUploadedTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "prepare task error")

		getResultInput := &inference_tasks.GetResultInput{
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
}

func TestGetImage(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "prepare accounts error")

		_, task, err := tests.PrepareResultUploadedTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "prepare task error")

		var imageNum, srcFile, dstFile string

		if taskType == models.TaskTypeSD {
			imageNum = "2"
			srcFile = "2.png"
			dstFile = "downloaded.png"
		} else {
			imageNum = "0"
			srcFile = "0.json"
			dstFile = "downloaded.json"
		}

		getResultInput := &inference_tasks.GetResultInput{
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

		assert.Equal(t, 200, r.Code, "wrong http status code. message: "+string(r.Body.Bytes()))

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

		t.Cleanup(func() {
			tests.ClearDB()
			if err := tests.ClearDataFolders(); err != nil {
				t.Error(err)
			}
		})
	}
}

func callGetImageApi(
	taskIdStr string,
	imageNum string,
	timestamp int64,
	signature string) *httptest.ResponseRecorder {

	endpoint := "/v1/inference_tasks/" + taskIdStr + "/results/" + imageNum
	query := "?timestamp=" + strconv.FormatInt(timestamp, 10) + "&signature=" + signature

	req, _ := http.NewRequest("GET", endpoint+query, nil)
	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}
