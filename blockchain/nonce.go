package blockchain

import (
	"container/heap"
	"context"
	"math/big"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common"
)

type NonceHeap []*big.Int

func (h NonceHeap) Len() int { return len(h) }
func (h NonceHeap) Less(i, j int) bool {
	return h[i].Cmp(h[j]) < 0
}
func (h NonceHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *NonceHeap) Push(x any) {
	*h = append(*h, x.(*big.Int))
}

func (h *NonceHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type NonceKeeper map[common.Address]*NonceHeap

var nonceKeeper NonceKeeper = make(NonceKeeper)
var nonceMutex sync.Mutex

func getNonce(ctx context.Context, address common.Address) (*big.Int, error) {
	nonceMutex.Lock()
	defer nonceMutex.Unlock()
	if _, ok := nonceKeeper[address]; !ok {
		client, err := GetRpcClient()
		if err != nil {
			return nil, nil
		}

		if err := getLimiter().Wait(ctx); err != nil {
			return nil, nil
		}

		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		nonce, err := client.PendingNonceAt(callCtx, address)
		if err != nil {
			return nil, nil
		}
		log.Debugln("Nonce from blockchain: " + strconv.FormatUint(nonce, 10))
		h := &NonceHeap{big.NewInt(int64(nonce))}
		heap.Init(h)
		nonceKeeper[address] = h
	}
	nonce := heap.Pop(nonceKeeper[address]).(*big.Int)
	log.Infof("Nonce: Get nonce: %d", nonce.Uint64())
	heap.Push(nonceKeeper[address], big.NewInt(0).Add(nonce, big.NewInt(1)))
	return nonce, nil
}

func restoreNonce(address common.Address, nonce *big.Int) {
	if nonce != nil {
		heap.Push(nonceKeeper[address], nonce)
		log.Infof("Nonce: Restore nonce: %d", nonce.Uint64())
	}
}
