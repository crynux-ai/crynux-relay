package inference_tasks

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/storyicon/sigverify"
	"math"
	"strconv"
	"time"
)

func ValidateSignature(address string, data []byte, timestamp int64, signature string) (bool, error) {

	current := time.Now().Unix()

	if math.Abs(float64(current-timestamp)) > 60 {
		return false, nil
	}

	timeStr := strconv.FormatInt(timestamp, 10)

	timeByte := []byte(timeStr)

	signBytes := append(data, timeByte...)

	return sigverify.VerifyEllipticCurveHexSignatureEx(
		ethcommon.HexToAddress(address),
		signBytes,
		signature,
	)
}
