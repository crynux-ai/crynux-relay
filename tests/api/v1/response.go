package v1

import (
	"encoding/json"
	"github.com/magiconair/properties/assert"
	"h_relay/api/v1/response"
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
