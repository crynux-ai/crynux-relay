package v1

import (
	"crypto/ecdsa"
	"errors"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"strconv"
	"time"
)

func SignData(data []byte, privateKeyStr string) (timestamp int64, signature string, err error) {

	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return 0, "", err
	}

	timestamp = time.Now().Unix()
	timestampStr := strconv.FormatInt(timestamp, 10)

	dataHash := crypto.Keccak256Hash(data, []byte(timestampStr))

	signBytes := append(dataHash.Bytes())

	signatureBytes, err := crypto.Sign(signBytes, privateKey)
	if err != nil {
		return 0, "", err
	}

	signature = hexutil.Encode(signatureBytes)

	return timestamp, signature, nil
}

func CreateAccount() (address string, privateKeyStr string, err error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return "", "", err
	}

	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyStr = hexutil.Encode(privateKeyBytes)
	privateKeyStr = privateKeyStr[2:] // remove the heading 0x

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", "", errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	address = crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	return address, privateKeyStr, nil
}
