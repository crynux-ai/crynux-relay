package v1

import (
	"encoding/json"
	"github.com/magiconair/properties/assert"
	"h_relay/api/v1/inference_tasks"
	"h_relay/api/v1/response"
	"h_relay/models"
	"net/http/httptest"
	"testing"
)

func AssertValidationErrorResponse(t *testing.T, r *httptest.ResponseRecorder, fieldName, fieldMessage string) {

	assert.Equal(t, r.Code, 400)

	// Deserialize response message

	validationResponse := &response.ValidationErrorResponse{}

	err := json.Unmarshal(r.Body.Bytes(), validationResponse)
	assert.Equal(t, err, nil)

	assert.Equal(t, validationResponse.GetErrorType(), "validation_error")
	assert.Equal(t, validationResponse.GetFieldName(), fieldName)
	assert.Equal(t, validationResponse.GetFieldMessage(), fieldMessage)
}

func AssertExceptionResponse(t *testing.T, r *httptest.ResponseRecorder, message string) {
	assert.Equal(t, r.Code, 500)

	exceptionResponse := &response.ExceptionResponse{}

	err := json.Unmarshal(r.Body.Bytes(), exceptionResponse)
	assert.Equal(t, err, nil)

	assert.Equal(t, exceptionResponse.GetMessage(), message)
}

func AssertTaskResponse(t *testing.T, r *httptest.ResponseRecorder, task *models.InferenceTask) {
	assert.Equal(t, r.Code, 200, "wrong http status code")

	taskResponse := &inference_tasks.TaskResponse{}

	responseBytes := r.Body.Bytes()

	err := json.Unmarshal(responseBytes, taskResponse)
	assert.Equal(t, err, nil, "json unmarshal error")

	assert.Equal(t, taskResponse.GetMessage(), "success", "wrong message: "+string(responseBytes))
	assert.Equal(t, taskResponse.Data.TaskId, task.TaskId, "wrong task id")
	assert.Equal(t, taskResponse.Data.TaskParams, task.TaskParams, "wrong task params")
	assert.Equal(t, taskResponse.Data.CreatedAt.IsZero(), false, "wrong task created at")
	assert.Equal(t, taskResponse.Data.UpdatedAt.IsZero(), false, "wrong task updated at")
}
