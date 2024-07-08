package blockchain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"

)

func GetAllNodesNumber() (busyNodes *big.Int, allNodes *big.Int, activeNodes *big.Int, err error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), err
	}

	allNodes, err = netstatsInstance.TotalNodes(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), err
	}

	busyNodes, err = netstatsInstance.BusyNodes(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), err
	}

	activeNodes, err = netstatsInstance.ActiveNodes(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0), err
	}

	return busyNodes, allNodes, activeNodes, nil
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
	Address   string   `json:"address"`
	CardModel string   `json:"card_model"`
	VRam      int      `json:"v_ram"`
	Balance   *big.Int `json:"balance"`
	Active    bool     `json:"active"`
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

	nodeContractInstance, err := GetNodeContractInstance()
	if err != nil {
		return nil, err
	}

	allNodeInfos, err := netstatsInstance.GetAllNodeInfo(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	}, big.NewInt(int64(startIndex)), big.NewInt(int64(endIndex-startIndex)))
	if err != nil {
		return nil, err
	}

	nodeData := make([]NodeData, len(allNodeInfos))

	errCh := make(chan error, len(allNodeInfos))
	// limit concurrency goroutines count
	limiter := make(chan struct{}, 4)

	for idx, nodeInfo := range allNodeInfos {
		nodeData[idx] = NodeData{
			Address:   nodeInfo.NodeAddress.Hex(),
			CardModel: nodeInfo.GPUModel,
			VRam:      int(nodeInfo.VRAM.Int64()),
			Balance: big.NewInt(0),
		}

		limiter <- struct{}{}
		go func(idx int, nodeAddress common.Address) {

			defer func() {
				<-limiter
			}()

			balance, err := client.BalanceAt(context.Background(), nodeAddress, nil)

			if err != nil {
				log.Errorf("get wallet balance error: %v", err)
				errCh <- err
				return
			}

			status, err := nodeContractInstance.GetNodeStatus(&bind.CallOpts{
				Pending: false,
				Context: context.Background(),
			}, nodeAddress)
			if err != nil {
				log.Errorf("get node status error: %v", err)
				errCh <- err
				return
			}

			if status.Cmp(big.NewInt(0)) != 0 {
				nodeData[idx].Active = true
			}

			nodeData[idx].Balance = balance
			errCh <- nil
		}(idx, nodeInfo.NodeAddress)
	}

	for i := 0; i < len(allNodeInfos); i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}
	close(errCh)

	return nodeData, nil
}
