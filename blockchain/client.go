package blockchain

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"h_relay/config"
)

var ethWSClient *ethclient.Client

func GetWebSocketClient() (*ethclient.Client, error) {

	if ethWSClient == nil {
		appConfig := config.GetConfig()
		client, err := ethclient.Dial(appConfig.Blockchain.WebSocketEndpoint)

		if err != nil {
			return nil, err
		}

		ethWSClient = client
	}

	return ethWSClient, nil
}
