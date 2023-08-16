package blockchain

import (
	"github.com/ethereum/go-ethereum/common"
	"h_relay/blockchain/bindings"
	"h_relay/config"
)

var taskContractInstance *bindings.Task

func GetTaskContractInstance() (*bindings.Task, error) {

	if taskContractInstance == nil {
		appConfig := config.GetConfig()
		taskContractAddress := common.HexToAddress(appConfig.Blockchain.Contracts.Task)

		client, err := GetWebSocketClient()
		if err != nil {
			return nil, err
		}

		instance, err := bindings.NewTask(taskContractAddress, client)

		if err != nil {
			return nil, err
		}

		taskContractInstance = instance
	}

	return taskContractInstance, nil
}
