package v1

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"h_relay/models"
)

func PrepareAccounts() (addresses []string, privateKeys []string, err error) {

	for i := 0; i < 5; i++ {
		address, pk, err := CreateAccount()
		if err != nil {
			return nil, nil, err
		}

		addresses = append(addresses, address)
		privateKeys = append(privateKeys, pk)
	}

	log.Debugln(addresses)
	log.Debugln(privateKeys)

	return addresses, privateKeys, nil
}

func PrepareTask(addresses []string) (*models.InferenceTask, error) {

	selectedNodes := [3]string{
		addresses[1],
		addresses[2],
		addresses[3],
	}

	selectedNodesStr, err := json.Marshal(selectedNodes)
	if err != nil {
		return nil, err
	}

	task := &models.InferenceTask{
		TaskId:        199,
		Creator:       addresses[0],
		TaskParams:    "{\"steps\":40}",
		SelectedNodes: string(selectedNodesStr),
	}

	return task, nil
}
