package task_args_test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"h_relay/models/task_args"
	"h_relay/tests"
	"testing"
)

func TestFullArgsJSON(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgsBytes, err := json.Marshal(taskArgs)
	assert.Nil(t, err, "json marshall error")
	assert.Equal(t, tests.FullTaskArgsString(), string(taskArgsBytes))
}

func TestNullControlnetArgsJSON(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Controlnet = nil
	transformAndCompareJSON(t, taskArgs)
}

func TestNullControlnetPreprocessArgsJSON(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Controlnet.Preprocess = nil
	transformAndCompareJSON(t, taskArgs)
}

func TestNullControlnetPreprocessMethodArgsJSON(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Controlnet.Preprocess.Args = nil
	transformAndCompareJSON(t, taskArgs)
}

func TestNullLoraArgsJSON(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Lora = nil
	transformAndCompareJSON(t, taskArgs)
}

func TestNullRefinerArgsJSON(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Refiner = nil
	transformAndCompareJSON(t, taskArgs)
}

func transformAndCompareJSON(t *testing.T, taskArgs *task_args.TaskArgs) {
	jsonBytes, err := json.Marshal(taskArgs)
	assert.Nil(t, err, "json marshall error")

	transformedTaskArgs := &task_args.TaskArgs{}

	err = json.Unmarshal(jsonBytes, transformedTaskArgs)
	assert.Nil(t, err, "json unmarshall error")

	transformedTaskArgsBytes, err := json.Marshal(transformedTaskArgs)
	assert.Nil(t, err, "json unmarshall error")

	assert.Equal(t, string(jsonBytes), string(transformedTaskArgsBytes), "args mismatch")
}
