package inference_tasks_test

import (
	"crynux_relay/api/v1/inference_tasks"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/tests"
	v1 "crynux_relay/tests/api/v1"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrongTaskId(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		addresses, privateKeys, err := tests.PrepareAccounts()
		assert.Equal(t, nil, err, "prepare accounts error")

		_, _, err = tests.PreparePendingResultsTask(taskType, addresses, config.GetDB())
		assert.Equal(t, nil, err, "prepare task error")

		uploadResultInput := &inference_tasks.ResultInput{
			TaskId: 666,
		}

		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		timestamp, signature, err := v1.SignData(uploadResultInput, privateKeys[1])
		assert.Equal(t, nil, err, "sign data error")

		prepareFileForm(t, writer, taskType, timestamp, signature)

		r := callUploadResultApi(666, writer, pr)

		v1.AssertValidationErrorResponse(t, r, "task_id", "Task not found")

		t.Cleanup(func() {
			tests.ClearDB()
			if err := tests.ClearDataFolders(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestCreatorUpload(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		testUsingAddressNum(t, 0, taskType, func(
			t *testing.T,
			r *httptest.ResponseRecorder,
			task *models.InferenceTask,
			addresses []string) {
			v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")
		})
	}
}

func TestNotAllowedAccount(t *testing.T) {
	for _, taskType := range tests.TaskTypes {
		testUsingAddressNum(t, 4, taskType, func(
			t *testing.T,
			r *httptest.ResponseRecorder,
			task *models.InferenceTask,
			addresses []string) {
			v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")
		})
	}
}

func TestSuccessfulUpload(t *testing.T) {
	testUsingAddressNum(t, 2, models.TaskTypeSD, func(
		t *testing.T,
		r *httptest.ResponseRecorder,
		task *models.InferenceTask,
		addresses []string) {

		v1.AssertEmptySuccessResponse(t, r)

		for i := 0; i < 5; i++ {
			assertFileExists(t, task.TaskId, models.TaskTypeSD, i)
		}
	})

	testUsingAddressNum(t, 2, models.TaskTypeLLM, func(
		t *testing.T,
		r *httptest.ResponseRecorder,
		task *models.InferenceTask,
		addresses []string) {

		v1.AssertEmptySuccessResponse(t, r)

		assertFileExists(t, task.TaskId, models.TaskTypeLLM, 0)
	})

}

func testUsingAddressNum(
	t *testing.T,
	num int,
	taskType models.ChainTaskType,
	assertFunc func(
		t *testing.T,
		r *httptest.ResponseRecorder,
		task *models.InferenceTask,
		addresses []string)) {

	addresses, privateKeys, err := tests.PrepareAccounts()
	assert.Equal(t, nil, err, "prepare accounts error")

	_, task, err := tests.PreparePendingResultsTask(taskType, addresses, config.GetDB())
	assert.Equal(t, nil, err, "prepare task error")

	assert.Equal(t, models.InferenceTaskPendingResults, task.Status, "wrong task status")

	uploadResultInput := &inference_tasks.ResultInput{
		TaskId: task.TaskId,
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	timestamp, signature, err := v1.SignData(uploadResultInput, privateKeys[num])
	assert.Equal(t, nil, err, "sign data error")

	prepareFileForm(t, writer, taskType, timestamp, signature)

	r := callUploadResultApi(task.TaskId, writer, pr)

	assertFunc(t, r, task, addresses)

	t.Cleanup(func() {
		tests.ClearDB()
		if err := tests.ClearDataFolders(); err != nil {
			t.Error(err)
		}
	})

}

func prepareFileForm(t *testing.T, writer *multipart.Writer, taskType models.ChainTaskType, timestamp int64, signature string) {
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

		if taskType == models.TaskTypeSD {
			for i := 0; i < 9; i++ {
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
		} else {
			part, err := writer.CreateFormFile("images", "test_resp.json")
			if err != nil {
				t.Error(err)
			}

			_, err = part.Write([]byte(tests.GPTResponseStr))
			if err != nil {
				t.Error(err)
			}
		}

	}()
}

func assertFileExists(t *testing.T, taskId uint64, taskType models.ChainTaskType, imageNum int) {
	taskIdStr := strconv.FormatUint(taskId, 10)
	var ext string
	if taskType == models.TaskTypeSD {
		ext = ".png"
	} else {
		ext = ".json"
	}
	imageFilename := strconv.Itoa(imageNum) + ext

	appConfig := config.GetConfig()
	imageFilePath := filepath.Join(appConfig.DataDir.InferenceTasks, taskIdStr, "results", imageFilename)

	_, err := os.Stat(imageFilePath)

	assert.Equal(t, nil, err, "image not exist")
}

func callUploadResultApi(taskId uint64, writer *multipart.Writer, pr *io.PipeReader) *httptest.ResponseRecorder {
	taskIdStr := strconv.FormatUint(taskId, 10)

	req, _ := http.NewRequest("POST", "/v1/inference_tasks/"+taskIdStr+"/results", pr)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	tests.Application.ServeHTTP(w, req)

	return w
}
