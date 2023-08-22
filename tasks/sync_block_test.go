package tasks_test

import (
	"github.com/stretchr/testify/assert"
	"h_relay/blockchain"
	"h_relay/config"
	"h_relay/models"
	"h_relay/tests"
	v1 "h_relay/tests/api/v1"
	"testing"
	"time"
)

func TestTaskCreatedOnChain(t *testing.T) {
	addresses, privateKeys, err := tests.PrepareAccountsWithTokens()
	assert.Nil(t, err, "error preparing accounts")

	err = tests.PrepareNetwork(addresses, privateKeys)
	assert.Nil(t, err, "error preparing network nodes")

	err = tests.PrepareTaskCreatorAccount(addresses[0], privateKeys[0])
	assert.Nil(t, err, "error preparing task creator account")

	appConfig := config.GetConfig()
	appConfig.Blockchain.Account.Address = addresses[0]
	appConfig.Blockchain.Account.PrivateKey = privateKeys[0]

	taskInput := v1.PrepareRandomTask()

	task := &models.InferenceTask{
		Prompt:     taskInput.Prompt,
		BaseModel:  taskInput.BaseModel,
		LoraModel:  taskInput.LoraModel,
		TaskConfig: &taskInput.TaskConfig,
		Pose:       &taskInput.Pose,
	}

	_, err = blockchain.CreateTaskOnChain(task)
	assert.Nil(t, err, "error creating task on chain")

	time.Sleep(20 * time.Second)

}
