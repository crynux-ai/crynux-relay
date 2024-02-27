package blockchain

import (
	"context"
	"crynux_relay/blockchain/bindings"
	"crynux_relay/config"
	"crynux_relay/models"
	"crynux_relay/utils"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"image/png"
	"io"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
)

func CreateTaskOnChain(task *models.InferenceTask) (string, error) {

	appConfig := config.GetConfig()

	taskHash, err := task.GetTaskHash()
	if err != nil {
		return "", err
	}

	taskContractAddress := common.HexToAddress(appConfig.Blockchain.Contracts.Task)
	accountAddress := common.HexToAddress(appConfig.Blockchain.Account.Address)
	accountPrivateKey := appConfig.Blockchain.Account.PrivateKey

	client, err := GetRpcClient()
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

	dataHash := [32]byte{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0}

	tx, err := instance.CreateTask(auth, big.NewInt(int64(task.TaskType)), *taskHash, dataHash, big.NewInt(int64(task.VramLimit)))
	if err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}

func GetTaskCreationResult(txHash string) (*big.Int, error) {

	client, err := GetRpcClient()
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

func GetTaskById(taskId uint64) (*bindings.TaskInfo, error) {

	taskInstance, err := GetTaskContractInstance()
	if err != nil {
		return nil, err
	}

	opts := &bind.CallOpts{
		Pending: false,
		Context: context.Background(),
	}

	taskInfo, err := taskInstance.GetTask(opts, big.NewInt(int64(taskId)))
	if err != nil {
		return nil, err
	}

	return &taskInfo, nil
}

func GetPHashForImage(reader io.Reader) ([]byte, error) {
	image, err := png.Decode(reader)
	if err != nil {
		return nil, err
	}
	pHash, err := goimagehash.PerceptionHash(image)
	if err != nil {
		return nil, err
	}

	bs := make([]byte, pHash.Bits()/8)
	binary.BigEndian.PutUint64(bs, pHash.GetHash())
	return bs, nil
}

func GetHashForGPTResponse(response interface{}) ([]byte, error) {
	content, err := utils.JSONMarshalWithSortedKeys(response)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(content)
	return h[:], nil
}
