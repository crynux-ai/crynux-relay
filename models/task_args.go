package models

import (
	"crynux_relay/config"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader"
)

var sdInferenceTaskSchema *jsonschema.Schema
var gptInferenceTaskSchema *jsonschema.Schema
var sdFinetuneLoraTaskSchema *jsonschema.Schema

func ValidateTaskArgsJsonStr(jsonStr string, taskType TaskType) (validationError, err error) {
	if taskType == TaskTypeSD {
		return validateSDTaskArgs(jsonStr)
	} else if taskType == TaskTypeLLM {
		return validateGPTTaskArgs(jsonStr)
	} else {
		return validateSDFinetuneLoraTaskArgs(jsonStr)
	}
}

func validateSDTaskArgs(jsonStr string) (validationError, err error) {
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

func validateGPTTaskArgs(jsonStr string) (validationError, err error) {
	if gptInferenceTaskSchema == nil {
		schemaJson := config.GetConfig().TaskSchema.GPTInference

		if !isValidUrl(schemaJson) {
			return nil, errors.New("invalid URL for task json schema")
		}

		gptInferenceTaskSchema, err = jsonschema.Compile(schemaJson)

		if err != nil {
			return nil, err
		}
	}

	var v interface{}
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return nil, err
	}

	return gptInferenceTaskSchema.Validate(v), nil
}

func validateSDFinetuneLoraTaskArgs(jsonStr string) (validationError, err error) {
	if sdFinetuneLoraTaskSchema == nil {
		schemaJson := config.GetConfig().TaskSchema.StableDiffusionFinetuneLora

		if !isValidUrl(schemaJson) {
			return nil, errors.New("invalid URL for task json schema")
		}

		sdFinetuneLoraTaskSchema, err = jsonschema.Compile(schemaJson)

		if err != nil {
			return nil, err
		}
	}

	var v interface{}
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return nil, err
	}
	return sdFinetuneLoraTaskSchema.Validate(v), nil
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
