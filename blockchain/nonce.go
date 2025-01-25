package blockchain

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common"
)

var doneTxCount *uint64
var txMutex sync.Mutex

var pendingNonceTxs map[uint64]string = make(map[uint64]string)
var pendingTxNonce map[string]uint64 = make(map[string]uint64)

var pattern *regexp.Regexp = regexp.MustCompile(`invalid nonce; got (\d+), expected (\d+)`)

func getNonce(ctx context.Context, address common.Address) (uint64, error) {
	if doneTxCount == nil {
		client, err := GetRpcClient()
		if err != nil {
			return 0, err
		}

		if err := getLimiter().Wait(ctx); err != nil {
			return 0, err
		}

		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		nonce, err := client.PendingNonceAt(callCtx, address)
		if err != nil {
			return 0, err
		}
		log.Debugln("Nonce from blockchain: " + strconv.FormatUint(nonce, 10))
		doneTxCount = &nonce
	}
	return (*doneTxCount) + uint64(len(pendingTxNonce)), nil
}

func isTxPending(txHash string) bool {
	_, ok := pendingTxNonce[txHash]
	return ok
}

func addPendingTx(txHash string, nonce uint64) {
	if *doneTxCount != nonce {
		log.Panic(fmt.Sprintf("local nonce changed when add pending tx, local nonce: %d, coming nonce: %d", *doneTxCount, nonce))
	}
	pendingTxNonce[txHash] = nonce
	pendingNonceTxs[nonce] = txHash
}

func donePendingTx(txHash string) {
	if _, ok := pendingTxNonce[txHash]; !ok {
		log.Panic(fmt.Sprintf("tx %s is not pending, cannot be done", txHash))
	}
	nonce := pendingTxNonce[txHash]
	delete(pendingTxNonce, txHash)
	delete(pendingNonceTxs, nonce)
	(*doneTxCount)++
}

func cancelPendingTx(txHash string) {
	if _, ok := pendingTxNonce[txHash]; !ok {
		log.Panic(fmt.Sprintf("tx %s is not pending, cannot be cancelled", txHash))
	}
	nonce := pendingTxNonce[txHash]
	delete(pendingTxNonce, txHash)
	delete(pendingNonceTxs, nonce)
}

func resetNonce(nonce uint64) {
	*doneTxCount = nonce
	pendingCnt := len(pendingNonceTxs)
	for i := 0; i < pendingCnt; i++ {
		nextNonce := nonce + uint64(i)
		if txHash, ok := pendingNonceTxs[nextNonce]; ok {
			delete(pendingNonceTxs, nextNonce)
			delete(pendingTxNonce, txHash)
		}
	}
}

func matchNonceError(errStr string) (uint64, bool) {
	res := pattern.FindStringSubmatch(errStr)
	if res == nil {
		return 0, false
	}
	nonceStr := res[len(res) - 1]
	if len(nonceStr) == 0 {
		return 0, false
	}
	nonce, _ := strconv.ParseUint(nonceStr, 10, 64)
	return nonce, true
}

func processSendingTxError(err error) error {
	if nonce, ok := matchNonceError(err.Error()); ok {
		resetNonce(nonce)
	}
	return err
}