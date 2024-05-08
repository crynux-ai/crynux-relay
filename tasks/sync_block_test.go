package tasks_test

import (
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/tasks"
	"crynux_relay/tests"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func TestTaskCreatedAndSuccessOnChain(t *testing.T) {

	err := tests.SyncToLatestBlock()
	assert.Nil(t, err, "error syncing to the latest block")

	syncBlockChan := make(chan int)
	go tasks.StartSyncBlockWithTerminateChannel(syncBlockChan)

	addresses, privateKeys, err := tests.PrepareAccountsWithTokens()
	assert.Nil(t, err, "error preparing accounts")

	err = tests.PrepareNetwork(addresses, privateKeys)
	assert.Nil(t, err, "error preparing network nodes")

	err = tests.PrepareTaskCreatorAccount(addresses[0], privateKeys[0])
	assert.Nil(t, err, "error preparing task creator account")

	appConfig := config.GetConfig()
	appConfig.Blockchain.Account.Address = addresses[0]
	appConfig.Blockchain.Account.PrivateKey = privateKeys[0]

	for _, taskType := range tests.TaskTypes {
		taskInput, err := tests.PrepareRandomTask(taskType)
		assert.Nil(t, err, "error preparing random task")

		task := &models.InferenceTask{
			TaskArgs:  taskInput.TaskArgs,
			TaskType:  taskType,
			VramLimit: 8,
		}

		_, err = blockchain.CreateTaskOnChain(task)
		assert.Nil(t, err, "error creating task on chain")

		time.Sleep(5 * time.Second)

		taskInDb := &models.InferenceTask{}

		err = config.GetDB().Model(taskInDb).Last(taskInDb).Error
		assert.Nil(t, err, "task not created")

		// Task in DB has no params for now
		// The params will be uploaded by the task creator later

		taskHash, err := task.GetTaskHash()
		assert.Nil(t, err, "error getting task hash")

		assert.Equal(t, taskHash.Hex(), taskInDb.TaskHash, "task hash mismatch")

		// Now Let's finish the task on chain

		err = tests.SubmitResultOnChain(big.NewInt(int64(taskInDb.TaskId)), addresses, privateKeys)
		assert.Nil(t, err, "error submitting task result on chain")

		time.Sleep(5 * time.Second)

		taskInDbWithSelectedNodes := &models.InferenceTask{
			TaskId: taskInDb.TaskId,
		}

		err = config.GetDB().Where(taskInDbWithSelectedNodes).Preload("SelectedNodes").First(taskInDbWithSelectedNodes).Error
		assert.Nil(t, err, "error finding task in db")

		assert.Equal(t, models.InferenceTaskPendingResults, taskInDbWithSelectedNodes.Status, "Task not success on chain")

		assert.Equal(t, 3, len(taskInDbWithSelectedNodes.SelectedNodes), "wrong node number")

		targetHash := hexutil.Encode([]byte("12345678"))

		check := &models.SelectedNode{
			InferenceTaskID:  taskInDbWithSelectedNodes.ID,
			IsResultSelected: true,
			Result:           targetHash,
		}

		err = config.GetDB().Where(check).First(check).Error
		assert.Nil(t, err, "error find result success node")
	}

	t.Cleanup(func() {
		syncBlockChan <- 1
		err := tests.ClearNetwork(addresses, privateKeys)
		assert.Nil(t, err, "clear network error")
		tests.ClearDB()
	})
}
