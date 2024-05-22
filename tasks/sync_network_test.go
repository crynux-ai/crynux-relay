package tasks_test

import (
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/tasks"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncNetWork(t *testing.T) {
	err := tasks.SyncNetwork()
	if err != nil {
		t.Fatal(err)
	}

	var nodeNumber models.NetworkNodeNumber
	if err := config.GetDB().Model(&models.NetworkNodeNumber{}).First(&nodeNumber).Error; err != nil {
		t.Fatal(err)
	}
	assert.GreaterOrEqual(t, int(nodeNumber.AllNodes), 0)
	assert.GreaterOrEqual(t, int(nodeNumber.BusyNodes), 0)

	var taskNumber models.NetworkTaskNumber
	if err := config.GetDB().Model(&models.NetworkTaskNumber{}).First(&taskNumber).Error; err != nil {
		t.Fatal(err)
	}
	assert.GreaterOrEqual(t, int(taskNumber.TotalTasks), 0)

	step := 100
	var allNodeDatas []models.NetworkNodeData
	for start := 0; start < int(nodeNumber.AllNodes); start += step {
		var nodeDatas []models.NetworkNodeData
		if err := config.GetDB().Model(&models.NetworkNodeData{}).Order("id ASC").Limit(step).Offset(start).Find(&nodeDatas).Error; err != nil {
			t.Fatal(err)
		}
		allNodeDatas = append(allNodeDatas, nodeDatas...)
	}

	assert.Equal(t, len(allNodeDatas), int(nodeNumber.AllNodes))
	for _, nodeData := range allNodeDatas {
		res := nodeData.Balance.Cmp(big.NewInt(0))
		assert.Equal(t, res, 1)
	}
}