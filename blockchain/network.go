package blockchain

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"
)

func GetAllNodesNumber() (availableNode *big.Int, allNode *big.Int, err error) {
	nodeInstance, err := GetNodeContractInstance()
	if err != nil {
		return big.NewInt(0), big.NewInt(0), err
	}

	allNode, err = nodeInstance.TotalNodes(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), err
	}

	availableNode, err = nodeInstance.AvailableNodes(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})

	if err != nil {
		return big.NewInt(0), big.NewInt(0), err
	}

	return availableNode, allNode, nil
}

type NodeData struct {
	Address    string   `json:"address"`
	CardModel  string   `json:"card_model"`
	VRam       int      `json:"v_ram"`
	CNXBalance *big.Int `json:"cnx_balance"`
}

func GetAllNodesData(startIndex, endIndex int) ([]NodeData, error) {
	nodeInstance, err := GetNodeContractInstance()
	if err != nil {
		return nil, err
	}

	cnxInstance, err := GetCrynuxTokenContractInstance()
	if err != nil {
		return nil, err
	}

	allNode, err := nodeInstance.TotalNodes(&bind.CallOpts{
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

	allNodeAddresses, err := nodeInstance.GetAllNodeAddresses(&bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	})
	if err != nil {
		return nil, err
	}

	selectedNodeAddresses := allNodeAddresses[startIndex:endIndex]

	nodeData := make([]NodeData, len(selectedNodeAddresses))

	var wg sync.WaitGroup

	for idx, nodeAddress := range selectedNodeAddresses {
		wg.Add(1)

		go func(idx int, nodeAddress common.Address, nodeData []NodeData) {
			defer wg.Done()

			cnxBalance, err := cnxInstance.BalanceOf(&bind.CallOpts{
				Pending: false,
				Context: context.Background(),
			}, nodeAddress)

			if err != nil {
				return
			}

			nodeData[idx].Address = nodeAddress.Hex()
			nodeData[idx].CNXBalance = cnxBalance
			nodeData[idx].CardModel = "NVIDIA RTX 4090"
			nodeData[idx].VRam = 24

		}(idx, nodeAddress, nodeData)
	}

	wg.Wait()

	return nodeData, nil
}
