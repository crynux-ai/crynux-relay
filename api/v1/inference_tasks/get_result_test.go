package inference_tasks_test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"h_relay/api/v1/inference_tasks"
	"h_relay/config"
	"h_relay/tests"
	v1 "h_relay/tests/api/v1"
	"image/png"
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

	task, err := v1.PrepareTask(addresses)
	assert.Equal(t, nil, err, "prepare task error")

	err = config.GetDB().Create(task).Error
	assert.Equal(t, nil, err, "save task to db error")

	err = prepareImagesForNode(task.GetTaskIdAsString(), addresses[1])
	assert.Equal(t, nil, err, "create image error")

	getResultInput := &inference_tasks.GetResultInput{
		TaskId:       task.TaskId,
		SelectedNode: addresses[1],
		ImageNum:     0,
	}

	sigStr, err := json.Marshal(getResultInput)
	assert.Equal(t, nil, err, "json marshall error")

	timestamp, signature, err := v1.SignData(sigStr, privateKeys[1])
	assert.Equal(t, nil, err, "sign data error")

	r := callGetImageApi(
		task.GetTaskIdAsString(),
		addresses[1],
		0,
		timestamp,
		signature)

	v1.AssertValidationErrorResponse(t, r, "signature", "Signer not allowed")
}

func callGetImageApi(
	taskIdStr string,
	nodeAddress string,
	imageNum int,
	timestamp int64,
	signature string) *httptest.ResponseRecorder {

	endpoint := "/v1/inference_tasks/" + taskIdStr + "/results/" + nodeAddress + "/" + strconv.Itoa(imageNum)
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
