package task_args_test

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"h_relay/config"
	"h_relay/models"
	"h_relay/models/task_args"
	"h_relay/tests"
	"testing"
)

func TestFullArgsDB(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	inferenceTask := &models.InferenceTask{
		TaskId:   123,
		TaskArgs: *taskArgs,
	}

	err := config.GetDB().Save(inferenceTask).Error
	assert.Nil(t, err, "save to db failed")

	retrievedTask := &models.InferenceTask{
		TaskId: 123,
	}

	err = config.GetDB().Where(retrievedTask).First(retrievedTask).Error
	assert.Nil(t, err, "retrieve from db failed")

	retrievedTaskArgs := retrievedTask.TaskArgs

	retrievedTaskArgsBytes, err := json.Marshal(retrievedTaskArgs)
	assert.Nil(t, err, "json unmarshall error")

	assert.Equal(t, tests.FullTaskArgsString(), string(retrievedTaskArgsBytes), "args mismatch")

	t.Cleanup(tests.ClearDB)
}

func TestNullControlnetArgsDB(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Controlnet = nil
	saveAndCompareDB(t, taskArgs)
	t.Cleanup(tests.ClearDB)
}

func TestNullControlnetPreprocessArgsDB(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Controlnet.Preprocess = nil
	saveAndCompareDB(t, taskArgs)
	t.Cleanup(tests.ClearDB)
}

func TestNullControlnetPreprocessMethodArgsDB(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Controlnet.Preprocess.Args = nil
	saveAndCompareDB(t, taskArgs)
	t.Cleanup(tests.ClearDB)
}

func TestNullLoraArgsDB(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Lora = nil
	saveAndCompareDB(t, taskArgs)
	t.Cleanup(tests.ClearDB)
}

func TestNullRefinerArgsDB(t *testing.T) {
	taskArgs := tests.FullTaskArgs()
	taskArgs.Refiner = nil
	saveAndCompareDB(t, taskArgs)
	t.Cleanup(tests.ClearDB)
}

func saveAndCompareDB(t *testing.T, taskArgs *task_args.TaskArgs) {

	inferenceTask := &models.InferenceTask{
		TaskId:   123,
		TaskArgs: *taskArgs,
	}

	err := config.GetDB().Save(inferenceTask).Error
	assert.Nil(t, err, "save to db failed")

	retrievedTask := &models.InferenceTask{
		TaskId: 123,
	}

	err = config.GetDB().Where(retrievedTask).First(retrievedTask).Error
	assert.Nil(t, err, "retrieve from db failed")

	retrievedTaskArgs := retrievedTask.TaskArgs

	retrievedTaskArgsBytes, err := json.Marshal(retrievedTaskArgs)
	assert.Nil(t, err, "json unmarshall error")

	jsonBytes, err := json.Marshal(taskArgs)
	assert.Nil(t, err, "json marshall error")

	assert.Equal(t, string(jsonBytes), string(retrievedTaskArgsBytes), "args mismatch")
}
