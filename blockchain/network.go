package blockchain

import (
	"context"
	"math/big"
	"sync"
	"time"

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
	QoS       int64    `json:"qos"`
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

	qosContractInstance, err := GetQoSContractInstance()
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

	// limit concurrency goroutines count
	limiter := make(chan struct{}, 4)
	var wg sync.WaitGroup

	for idx, nodeInfo := range allNodeInfos {
		nodeData[idx] = NodeData{
			Address:   nodeInfo.NodeAddress.Hex(),
			CardModel: nodeInfo.GPUModel,
			VRam:      int(nodeInfo.VRAM.Int64()),
			Balance:   big.NewInt(0),
		}

		limiter <- struct{}{}
		wg.Add(1)
		go func(idx int, nodeAddress common.Address) {

			defer func() {
				<-limiter
				wg.Done()
			}()
			
			retryCount := 3

			for retryCount > 0 {
				balance, err := client.BalanceAt(context.Background(), nodeAddress, nil)
	
				if err != nil {
					log.Errorf("get wallet balance error: %v", err)
					retryCount -= 1
					time.Sleep(time.Second)
					continue
				}
				nodeData[idx].Balance = balance
				break
			}
	
			retryCount = 3
			for retryCount > 0{
				status, err := nodeContractInstance.GetNodeStatus(&bind.CallOpts{
					Pending: false,
					Context: context.Background(),
				}, nodeAddress)
				if err != nil {
					log.Errorf("get node status error: %v", err)
					retryCount -= 1
					time.Sleep(time.Second)
					continue
				}
	
				if status > 0 {
					nodeData[idx].Active = true
				}
				break
			}
	
			retryCount = 3
			for retryCount > 0 {
				qos, err := qosContractInstance.GetTaskScore(&bind.CallOpts{
					Pending: false,
					Context: context.Background(),
				}, nodeAddress)
				if err != nil {
					log.Errorf("get qos score error: %v", err)
					retryCount -= 1
					time.Sleep(time.Second)
					continue
				}
	
				nodeData[idx].QoS = qos.Int64()
				break
			}
		}(idx, nodeInfo.NodeAddress)
	}

	wg.Wait()

	return nodeData, nil
}
