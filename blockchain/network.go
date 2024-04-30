package blockchain

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func GetAllNodesNumber() (busyNodes *big.Int, allNodes *big.Int, err error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return big.NewInt(0), big.NewInt(0), err
	}

	allNodes, err = netstatsInstance.TotalNodes(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), err
	}

	busyNodes, err = netstatsInstance.BusyNodes(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), err
	}

	return busyNodes, allNodes, nil
}

func GetAllTasksNumber() (totalTasks *big.Int, runningTasks *big.Int, queuedTasks *big.Int, err error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), err
	}

	totalTasks, err = netstatsInstance.TotalTasks(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), err
	}

	runningTasks, err = netstatsInstance.RunningTasks(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), err
	}

	queuedTasks, err = netstatsInstance.QueuedTasks(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), err
	}

	return totalTasks, runningTasks, queuedTasks, nil
}

type NodeData struct {
	Address    string   `json:"address"`
	CardModel  string   `json:"card_model"`
	VRam       int      `json:"v_ram"`
	CNXBalance *big.Int `json:"cnx_balance"`
}

func GetAllNodesData(startIndex, endIndex int) ([]NodeData, error) {
	client, err := GetRpcClient()
		if err != nil {
			return nil, err
	}

	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return nil, err
	}

	allNode, err := netstatsInstance.TotalNodes(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return nil, err
	}

	if allNode.Cmp(big.NewInt(int64(startIndex))) <= 0 {
		return nil, errors.New("start index out of range")
	}

	if allNode.Cmp(big.NewInt(int64(endIndex))) < 0 {
		endIndex = int(allNode.Int64())
	}

	allNodeInfos, err := netstatsInstance.GetAllNodeInfo(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	}, big.NewInt(int64(startIndex)), big.NewInt(int64(endIndex-startIndex)))
	if err != nil {
		return nil, err
	}

	nodeData := make([]NodeData, len(allNodeInfos))

	var wg sync.WaitGroup

	for idx, nodeInfo := range allNodeInfos {
		wg.Add(1)

		nodeData[idx] = NodeData{
			Address:   nodeInfo.NodeAddress.Hex(),
			CardModel: nodeInfo.GPUModel,
			VRam:      int(nodeInfo.VRAM.Int64()),
		}

		go func(idx int, nodeAddress common.Address) {
			defer wg.Done()

			cnxBalance, err := client.BalanceAt(context.Background(), nodeAddress, nil)

			if err != nil {
				return
			}

			nodeData[idx].CNXBalance = cnxBalance
		}(idx, nodeInfo.NodeAddress)
	}

	wg.Wait()

	return nodeData, nil
}
