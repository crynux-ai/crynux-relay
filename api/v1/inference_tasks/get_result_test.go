package inference_tasks_test

import (
	"github.com/stretchr/testify/assert"
	"h_relay/api/v1/inference_tasks"
	"h_relay/config"
	"h_relay/tests"
	v1 "h_relay/tests/api/v1"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestUnauthorizedGetImage(t *testing.T) {
	addresses, privateKeys, err := v1.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare accounts error")

	_, task, err := v1.PrepareParamsUploadedTask(addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	err = prepareImagesForNode(task.GetTaskIdAsString(), addresses[1])
	assert.Equal(t, nil, err, "create image error")

	getResultInput := &inference_tasks.GetResultInput{
		TaskId:       task.TaskId,
		SelectedNode: addresses[1],
		ImageNum:     "0",
	}

	timestamp, signature, err := v1.SignData(getResultInput, privateKeys[1])
	assert.Equal(t, nil, err, "sign data error")

	r := callGetImageApi(
		task.GetTaskIdAsString(),
		addresses[1],
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

func TestGetImage(t *testing.T) {

	addresses, privateKeys, err := v1.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare accounts error")

	_, task, err := v1.PrepareParamsUploadedTask(addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	err = prepareImagesForNode(task.GetTaskIdAsString(), addresses[1])
	assert.Equal(t, nil, err, "create image error")

	getResultInput := &inference_tasks.GetResultInput{
		TaskId:       task.TaskId,
		SelectedNode: addresses[1],
		ImageNum:     "2",
	}

	timestamp, signature, err := v1.SignData(getResultInput, privateKeys[0])
	assert.Equal(t, nil, err, "sign data error")

	r := callGetImageApi(
		task.GetTaskIdAsString(),
		addresses[1],
		"2",
		timestamp,
		signature)

	assert.Equal(t, 200, r.Code, "wrong http status code. message: "+string(r.Body.Bytes()))

	appConfig := config.GetConfig()
	imageFolder := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.GetTaskIdAsString(),
		addresses[1],
	)

	out, err := os.Create(filepath.Join(imageFolder, "downloaded.png"))
	assert.Equal(t, nil, err, "create tmp file error")

	_, err = io.Copy(out, r.Body)
	assert.Equal(t, nil, err, "write tmp file error")

	err = out.Close()
	assert.Equal(t, nil, err, "close tmp file error")

	originalFile, err := os.Stat(filepath.Join(imageFolder, "2.png"))
	assert.Equal(t, nil, err, "read original file error")

	downloadedFile, err := os.Stat(filepath.Join(imageFolder, "downloaded.png"))
	assert.Equal(t, nil, err, "read downloaded file error")

	assert.Equal(t, originalFile.Size(), downloadedFile.Size(), "different file sizes")

	t.Cleanup(func() {
		tests.ClearDB()
		if err := tests.ClearDataFolders(); err != nil {
			t.Error(err)
		}
	})
}

func callGetImageApi(
	taskIdStr string,
	nodeAddress string,
	imageNum string,
	timestamp int64,
	signature string) *httptest.ResponseRecorder {

	endpoint := "/v1/inference_tasks/" + taskIdStr + "/results/" + nodeAddress + "/" + imageNum
	query := "?timestamp=" + strconv.FormatInt(timestamp, 10) + "&signature=" + signature

	req, _ := http.NewRequest("GET", endpoint+query, nil)
	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}

func prepareImagesForNode(taskIdStr, nodeAddress string) error {
	appConfig := config.GetConfig()

	imageFolder := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		taskIdStr,
		nodeAddress,
	)

	if err := os.MkdirAll(imageFolder, os.ModeDir); err != nil {
		return err
	}

	for i := 0; i < 5; i++ {
		filename := filepath.Join(imageFolder, strconv.Itoa(i)+".png")
		img := tests.CreateImage()
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			return err
		}

		if err := png.Encode(f, img); err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
