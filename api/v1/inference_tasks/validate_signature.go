package inference_tasks

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"math"
	"strconv"
	"time"
)

func ValidateSignature(data []byte, timestamp int64, signature string) (bool, string, error) {

	log.Debugln("signature to verify: " + signature)

	signatureBytes, err := hex.DecodeString(signature[2:])

	if err != nil {
		return false, "", err
	}

	current := time.Now().Unix()

	if math.Abs(float64(current-timestamp)) > 60 {
		return false, "", nil
	}

	timeStr := strconv.FormatInt(timestamp, 10)

	timeByte := []byte(timeStr)

	signBytes := append(data, timeByte...)

	log.Debugln("sign string: " + string(signBytes))

	dataHash := crypto.Keccak256Hash(signBytes)

	sigPublicKeyECDSA, err := crypto.SigToPub(dataHash.Bytes(), signatureBytes)

	if err != nil {
		return false, "", err
	}

	address := crypto.PubkeyToAddress(*sigPublicKeyECDSA).Hex()

	log.Debugln("address from signature: " + address)

	signatureNoRecoverID := signatureBytes[:len(signatureBytes)-1] // remove recovery ID

	verified := crypto.VerifySignature(
		crypto.FromECDSAPub(sigPublicKeyECDSA),
		dataHash.Bytes(),
		signatureNoRecoverID)

	return verified, address, nil
}
