package inference_tasks_test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"h_relay/api/v1/inference_tasks"
	"h_relay/config"
	"h_relay/models"
	"h_relay/tests"
	v1 "h_relay/tests/api/v1"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestWrongTaskId(t *testing.T) {
	addresses, privateKeys, err := v1.PrepareAccounts()
	assert.Equal(t, err, nil, "prepare accounts error")

	task, err := v1.PrepareTask(addresses)
	assert.Equal(t, err, nil, "prepare task error")

	err = config.GetDB().Create(task).Error
	assert.Equal(t, err, nil, "save task to db error")

	uploadResultInput := &inference_tasks.ResultInput{
		TaskId: 999,
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	signBytes, err := json.Marshal(uploadResultInput)
	assert.Equal(t, nil, err, "result input json marshall error")

	timestamp, signature, err := v1.SignData(signBytes, privateKeys[1])
	assert.Equal(t, nil, err, "sign data error")

	prepareFileForm(t, writer, timestamp, signature)

	r := callUploadResultApi(999, writer, pr)

	v1.AssertValidationErrorResponse(t, r, "task_id", "Task not found")

	t.Cleanup(func() {
		tests.ClearDB()
		if err := tests.ClearDataFolders(); err != nil {
			t.Error(err)
		}
	})
}

func TestCreatorUpload(t *testing.T) {
	testUsingAddressNum(t, 0, func(
		t *testing.T,
		r *httptest.ResponseRecorder,
		task *models.InferenceTask,
		addresses []string) {
		v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")
	})
}

func TestNotAllowedAccount(t *testing.T) {
	testUsingAddressNum(t, 4, func(
		t *testing.T,
		r *httptest.ResponseRecorder,
		task *models.InferenceTask,
		addresses []string) {
		v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")
	})
}

func TestSuccessfulUpload(t *testing.T) {
	testUsingAddressNum(t, 2, func(
		t *testing.T,
		r *httptest.ResponseRecorder,
		task *models.InferenceTask,
		addresses []string) {

		v1.AssertEmptySuccessResponse(t, r)

		for i := 0; i < 5; i++ {
			assertFileExists(t, task.TaskId, addresses[2], i)
		}
	})
}

func testUsingAddressNum(
	t *testing.T,
	num int,
	assertFunc func(
		t *testing.T,
		r *httptest.ResponseRecorder,
		task *models.InferenceTask,
		addresses []string)) {

	addresses, privateKeys, err := v1.PrepareAccounts()
	assert.Equal(t, err, nil, "prepare accounts error")

	task, err := v1.PrepareTask(addresses)
	assert.Equal(t, err, nil, "prepare task error")

	err = config.GetDB().Create(task).Error
	assert.Equal(t, err, nil, "save task to db error")

	uploadResultInput := &inference_tasks.ResultInput{
		TaskId: task.TaskId,
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	signBytes, err := json.Marshal(uploadResultInput)
	assert.Equal(t, nil, err, "result input json marshall error")

	timestamp, signature, err := v1.SignData(signBytes, privateKeys[num])
	assert.Equal(t, nil, err, "sign data error")

	prepareFileForm(t, writer, timestamp, signature)

	r := callUploadResultApi(task.TaskId, writer, pr)

	assertFunc(t, r, task, addresses)

	t.Cleanup(func() {
		tests.ClearDB()
		if err := tests.ClearDataFolders(); err != nil {
			t.Error(err)
		}
	})
}

func prepareFileForm(t *testing.T, writer *multipart.Writer, timestamp int64, signature string) {
	go func() {
		defer func(writer *multipart.Writer) {
			err := writer.Close()
			if err != nil {
				t.Error(err)
			}
		}(writer)

		timestampStr := strconv.FormatInt(timestamp, 10)

		err := writer.WriteField("timestamp", timestampStr)
		assert.Equal(t, nil, err, "write timestamp failed")

		err = writer.WriteField("signature", signature)
		assert.Equal(t, nil, err, "write signature failed")

		for i := 0; i < 5; i++ {
			part, err := writer.CreateFormFile("images", "test_image_"+strconv.Itoa(i)+".png")
			if err != nil {
				t.Error(err)
			}

			img := tests.CreateImage()

			if err != nil {
				t.Error(err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}
	}()
}

func assertFileExists(t *testing.T, taskId int64, selectedNode string, imageNum int) {
	taskIdStr := strconv.FormatInt(taskId, 10)
	imageFilename := strconv.Itoa(imageNum) + ".png"

	appConfig := config.GetConfig()
	imageFilePath := filepath.Join(appConfig.DataDir.InferenceTasks, taskIdStr, selectedNode, imageFilename)

	_, err := os.Stat(imageFilePath)

	assert.Equal(t, nil, err, "image not exist")
}

func callUploadResultApi(taskId int64, writer *multipart.Writer, pr *io.PipeReader) *httptest.ResponseRecorder {

	taskIdStr := strconv.FormatInt(taskId, 10)

	req, _ := http.NewRequest("POST", "/v1/inference_tasks/"+taskIdStr+"/results", pr)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}
