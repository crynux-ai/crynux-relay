package models

import (
	"encoding/json"
	"errors"
	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
	"h_relay/config"
	"net/url"
)

var sdInferenceTaskSchema *jsonschema.Schema

func ValidateTaskArgsJsonStr(jsonStr string) (validationError, err error) {

	if sdInferenceTaskSchema == nil {
		schemaJson := config.GetConfig().TaskSchema.StableDiffusionInference

		if !isValidUrl(schemaJson) {
			return nil, errors.New("invalid URL for task json schema")
		}

		sdInferenceTaskSchema, err = jsonschema.Compile(schemaJson)

		if err != nil {
			return nil, err
		}
	}

	var v interface{}
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return nil, err
	}

	return sdInferenceTaskSchema.Validate(v), nil
}

func isValidUrl(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}
