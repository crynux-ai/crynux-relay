package blockchain

import (
	"context"
	"crynux_relay/blockchain/bindings"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	log "github.com/sirupsen/logrus"
)

func GetTotalNodes(ctx context.Context) (*big.Int, error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := &bind.CallOpts{
		Pending: false,
		Context: callCtx,
	}

	if err := getLimiter().Wait(callCtx); err != nil {
		return nil, err
	}

	return netstatsInstance.TotalNodes(opts)
}

func GetBusyNodes(ctx context.Context) (*big.Int, error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := &bind.CallOpts{
		Pending: false,
		Context: callCtx,
	}

	if err := getLimiter().Wait(callCtx); err != nil {
		return nil, err
	}

	return netstatsInstance.BusyNodes(opts)
}

func GetActiveNodes(ctx context.Context) (*big.Int, error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := &bind.CallOpts{
		Pending: false,
		Context: callCtx,
	}

	if err := getLimiter().Wait(callCtx); err != nil {
		return nil, err
	}

	return netstatsInstance.ActiveNodes(opts)
}

func GetTotalTasks(ctx context.Context) (*big.Int, error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := &bind.CallOpts{
		Pending: false,
		Context: callCtx,
	}

	if err := getLimiter().Wait(callCtx); err != nil {
		return nil, err
	}

	return netstatsInstance.TotalTasks(opts)
}

func GetRunningTasks(ctx context.Context) (*big.Int, error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := &bind.CallOpts{
		Pending: false,
		Context: callCtx,
	}

	if err := getLimiter().Wait(callCtx); err != nil {
		return nil, err
	}

	return netstatsInstance.RunningTasks(opts)
}

func GetQueuedTasks(ctx context.Context) (*big.Int, error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := &bind.CallOpts{
		Pending: false,
		Context: callCtx,
	}

	if err := getLimiter().Wait(callCtx); err != nil {
		return nil, err
	}

	return netstatsInstance.QueuedTasks(opts)
}

func GetAllNodeInfo(ctx context.Context, offset, length *big.Int) ([]bindings.NetworkStatsNodeInfo, error) {
	netstatsInstance, err := GetNetstatsContractInstance()
	if err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := &bind.CallOpts{
		Pending: false,
		Context: callCtx,
	}

	if err := getLimiter().Wait(callCtx); err != nil {
		return nil, err
	}
	return netstatsInstance.GetAllNodeInfo(opts, offset, length)
}

func GetAllNodesNumber(ctx context.Context) (busyNodes *big.Int, allNodes *big.Int, activeNodes *big.Int, err error) {
	allNodes, err = GetTotalNodes(ctx)
	if err != nil {
		return
	}

	busyNodes, err = GetBusyNodes(ctx)
	if err != nil {
		return
	}

	activeNodes, err = GetActiveNodes(ctx)
	if err != nil {
		return
	}
	return busyNodes, allNodes, activeNodes, nil
}

func GetAllTasksNumber(ctx context.Context) (totalTasks *big.Int, runningTasks *big.Int, queuedTasks *big.Int, err error) {
	totalTasks, err = GetTotalTasks(ctx)
	if err != nil {
		return
	}

	runningTasks, err = GetRunningTasks(ctx)
	if err != nil {
		return
	}

	queuedTasks, err = GetQueuedTasks(ctx)
	if err != nil {
		return
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

func GetAllNodesData(ctx context.Context, startIndex, endIndex int) ([]NodeData, error) {
	allNodeInfos, err := GetAllNodeInfo(ctx, big.NewInt(int64(startIndex)), big.NewInt(int64(endIndex-startIndex)))
	if err != nil {
		return nil, err
	}

	nodeData := make([]NodeData, len(allNodeInfos))

	for idx, nodeInfo := range allNodeInfos {
		nodeData[idx] = NodeData{
			Address:   nodeInfo.NodeAddress.Hex(),
			CardModel: nodeInfo.GPUModel,
			VRam:      int(nodeInfo.VRAM.Int64()),
			Balance:   big.NewInt(0),
		}
		balance, err := BalanceAt(ctx, nodeInfo.NodeAddress)
		if err != nil {
			log.Errorf("GetAllNodesData: get wallet balance error: %v", err)
			return nil, err
		}
		nodeData[idx].Balance = balance

		status, err := GetNodeStatus(ctx, nodeInfo.NodeAddress)
		if err != nil {
			log.Errorf("GetAllNodesData: get node status error: %v", err)
			return nil, err
		}
		if status > 0 {
			nodeData[idx].Active = true
		}

		qos, err := GetTaskScore(ctx, nodeInfo.NodeAddress)
		if err != nil {
			log.Errorf("GetAllNodesData: get qos score error: %v", err)
			return nil, err
		}
		nodeData[idx].QoS = qos.Int64()
	}

	return nodeData, nil
}
