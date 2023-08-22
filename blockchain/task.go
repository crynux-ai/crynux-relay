package blockchain

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"h_relay/blockchain/bindings"
	"h_relay/config"
	"h_relay/models"
	"math/big"
	"math/rand"
	"strconv"
	"time"
)

func CreateTaskOnChain(task *models.InferenceTask) (string, error) {

	appConfig := config.GetConfig()

	taskHash, err := task.GetTaskHash()
	if err != nil {
		return "", err
	}

	dataHash, err := task.GetDataHash()
	if err != nil {
		return "", err
	}

	taskContractAddress := common.HexToAddress(appConfig.Blockchain.Contracts.Task)
	accountAddress := common.HexToAddress(appConfig.Blockchain.Account.Address)
	accountPrivateKey := appConfig.Blockchain.Account.PrivateKey

	client, err := GetWebSocketClient()
	if err != nil {
		return "", err
	}

	instance, err := bindings.NewTask(taskContractAddress, client)
	if err != nil {
		return "", err
	}

	auth, err := GetAuth(client, accountAddress, accountPrivateKey)
	if err != nil {
		return "", err
	}

	log.Debugln("create task tx: TaskHash " + common.Bytes2Hex(taskHash[:]))
	log.Debugln("create task tx: DataHash " + common.Bytes2Hex(dataHash[:]))

	tx, err := instance.CreateTask(auth, *taskHash, *dataHash)
	if err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}

func GetTaskCreationResult(txHash string) (*big.Int, error) {

	client, err := GetWebSocketClient()
	if err != nil {
		return nil, err
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), time.Duration(3)*time.Second)
	defer cancelFn()

	receipt, err := client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {

		if errors.Is(err, ethereum.NotFound) {
			// Transaction pending
			return nil, nil
		}

		log.Errorln("error getting tx receipt for: " + txHash)
		return nil, err
	}

	if receipt.Status == 0 {
		// Transaction failed
		// Get reason
		reason, err := GetErrorMessageForTxHash(receipt.TxHash, receipt.BlockNumber)

		if err != nil {
			log.Errorln("error getting error message for: " + txHash)
			return nil, err
		}

		return nil, errors.New(reason)
	}

	// Transaction success
	// Extract taskId from the logs
	taskContractInstance, err := GetTaskContractInstance()
	if err != nil {
		log.Errorln("error get task contract instance: " + receipt.TxHash.Hex())
		return nil, err
	}

	// There are 5 events emitted from the CreateTask method
	// Approval, Transfer, TaskCreated x 3
	if len(receipt.Logs) != 5 {
		log.Errorln(receipt.Logs)
		return nil, errors.New("wrong event logs number:" + strconv.Itoa(len(receipt.Logs)))
	}

	taskCreatedEvent, err := taskContractInstance.ParseTaskCreated(*receipt.Logs[2])
	if err != nil {
		log.Errorln("error parse task created event: " + receipt.TxHash.Hex())
		return nil, err
	}

	taskId := taskCreatedEvent.TaskId

	return taskId, nil
}

func GetTaskResultCommitment(result []byte) (commitment [32]byte, nonce [32]byte) {
	nonceStr := strconv.Itoa(rand.Int())
	nonceHash := crypto.Keccak256Hash([]byte(nonceStr))
	commitmentHash := crypto.Keccak256Hash(result, nonceHash.Bytes())
	copy(commitment[:], commitmentHash.Bytes())
	copy(nonce[:], nonceHash.Bytes())
	return commitment, nonce
}
