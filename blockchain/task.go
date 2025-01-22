package blockchain

import (
	"context"
	"crynux_relay/blockchain/bindings"
	"crynux_relay/config"
	"crypto/sha256"
	"encoding/binary"
	"image/png"
	"io"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func GetTaskByCommitment(ctx context.Context, taskIDCommitment [32]byte) (*bindings.VSSTaskTaskInfo, error) {
	taskInstance, err := GetTaskContractInstance()
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
	taskInfo, err := taskInstance.GetTask(opts, taskIDCommitment)
	if err != nil {
		return nil, err
	}

	return &taskInfo, nil
}

func ReportTaskParamsUploaded(ctx context.Context, taskIDCommitment [32]byte) (string, error) {
	taskInstance, err := GetTaskContractInstance()
	if err != nil {
		return "", err
	}

	appConfig := config.GetConfig()
	address := common.HexToAddress(appConfig.Blockchain.Account.Address)
	privkey := appConfig.Blockchain.Account.PrivateKey

	txMutex.Lock()
	defer txMutex.Unlock()

	auth, err := GetAuth(ctx, address, privkey)
	if err != nil {
		return "", err
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := getLimiter().Wait(callCtx); err != nil {
		return "", err
	}
	auth.Context = callCtx
	tx, err := taskInstance.ReportTaskParametersUploaded(auth, taskIDCommitment)
	if err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}

func ReportTaskResultUploaded(ctx context.Context, taskIDCommitment [32]byte) (string, error) {
	taskInstance, err := GetTaskContractInstance()
	if err != nil {
		return "", err
	}

	appConfig := config.GetConfig()
	address := common.HexToAddress(appConfig.Blockchain.Account.Address)
	privkey := appConfig.Blockchain.Account.PrivateKey

	txMutex.Lock()
	defer txMutex.Unlock()

	auth, err := GetAuth(ctx, address, privkey)
	if err != nil {
		return "", err
	}

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := getLimiter().Wait(callCtx); err != nil {
		return "", err
	}
	auth.Context = callCtx
	tx, err := taskInstance.ReportTaskResultUploaded(auth, taskIDCommitment)
	if err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
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

func GetHashForGPTResponse(reader io.Reader) ([]byte, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(content)
	return h[:], nil
}
