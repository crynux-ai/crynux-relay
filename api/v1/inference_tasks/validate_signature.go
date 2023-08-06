package inference_tasks

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/storyicon/sigverify"
)

func ValidateSignature(address string, data string, signature string) (bool, error) {
	return sigverify.VerifyEllipticCurveHexSignatureEx(
		ethcommon.HexToAddress(address),
		[]byte(data),
		signature,
	)
}
