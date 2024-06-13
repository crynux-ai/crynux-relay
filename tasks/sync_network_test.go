package tasks_test

import (
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/tasks"
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func getNodeNumber() (*models.NetworkNodeNumber, error) {
	var nodeNumber models.NetworkNodeNumber
	if err := config.GetDB().Model(&models.NetworkNodeNumber{}).First(&nodeNumber).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return &nodeNumber, nil
}

func getTaskNumber() (*models.NetworkTaskNumber, error) {
	var taskNumber models.NetworkTaskNumber
	if err := config.GetDB().Model(&models.NetworkTaskNumber{}).First(&taskNumber).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return &taskNumber, nil
}

func getNetworkFLOPS() (*models.NetworkFLOPS, error) {
	var flops models.NetworkFLOPS
	if err := config.GetDB().Model(&models.NetworkFLOPS{}).First(&flops).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return &flops, nil
}

func getAllNodeDatas(count int) ([]models.NetworkNodeData, error) {
	step := 100
	var allNodeDatas []models.NetworkNodeData
	for start := 0; start < count; start += step {
		var nodeDatas []models.NetworkNodeData
		if err := config.GetDB().Model(&models.NetworkNodeData{}).Order("id ASC").Limit(step).Offset(start).Find(&nodeDatas).Error; err != nil {
			return nil, err
		}
		allNodeDatas = append(allNodeDatas, nodeDatas...)
	}
	return allNodeDatas, nil
}

func TestSyncNetWork(t *testing.T) {
	nodeNumber, err := getNodeNumber()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, int(nodeNumber.AllNodes), 0)
	assert.Equal(t, int(nodeNumber.BusyNodes), 0)
	assert.Equal(t, int(nodeNumber.ActiveNodes), 0)

	taskNumber, err := getTaskNumber()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, int(taskNumber.TotalTasks), 0)

	flops, err := getNetworkFLOPS()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, int(flops.GFLOPS), 0)

	allNodeDatas, err := getAllNodeDatas(100)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(allNodeDatas), 0)

	err = tasks.SyncNetwork()
	if err != nil {
		t.Fatal(err)
	}

	nodeNumber, err = getNodeNumber()
	if err != nil {
		t.Fatal(err)
	}
	assert.GreaterOrEqual(t, int(nodeNumber.AllNodes), 0)
	assert.GreaterOrEqual(t, int(nodeNumber.BusyNodes), 0)
	assert.GreaterOrEqual(t, int(nodeNumber.ActiveNodes), 0)

	taskNumber, err = getTaskNumber()
	if err != nil {
		t.Fatal(err)
	}
	assert.GreaterOrEqual(t, int(taskNumber.TotalTasks), 0)

	flops, err = getNetworkFLOPS()
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, flops.GFLOPS, float64(0))

	allNodeDatas, err = getAllNodeDatas(int(nodeNumber.AllNodes))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(allNodeDatas), int(nodeNumber.AllNodes))
	for _, nodeData := range allNodeDatas {
		res := nodeData.Balance.Cmp(big.NewInt(0))
		assert.Equal(t, res, 1)
	}
}
