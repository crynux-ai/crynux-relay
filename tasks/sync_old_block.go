package tasks

import (
	"crynux_relay/blockchain"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

func processOldTxReceipt(txReceiptCh <-chan TxReceiptWithBlock) {
	for {
		receiptWithBlock, ok := <-txReceiptCh
		receipt := receiptWithBlock.TxReceipt
		block := receiptWithBlock.Block
		if !ok {
			break
		}

		for {
			log.Debugf("SyncedOldBlocks: processing task node success of %s", receipt.TxHash.Hex())
			if err := processTaskNodeSuccess(block, receipt); err != nil {
				log.Errorf("SyncedOldBlocks: processing task node success error: %v", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}
}

func syncOldBlocks(client *ethclient.Client, startBlock, endBlock uint64, concurrency int) {
	blocknumCh := make(chan uint64, 10)
	txHashCh := make(chan TxHashWithBlock, 10)
	txReceiptCh := make(chan TxReceiptWithBlock, 10)

	var blocknumWG sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		blocknumWG.Add(1)
		go func() {
			processBlocknum(client, blocknumCh, txHashCh)
			blocknumWG.Done()
		}()
	}

	go func() {
		blocknumWG.Wait()
		close(txHashCh)
		log.Debugf("SyncedOldBlocks: all blocknums %d:%d have been processed", startBlock, endBlock)
	}()

	var txHashWG sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		txHashWG.Add(1)
		go func() {
			processTxHash(client, txHashCh, txReceiptCh)
			txHashWG.Done()
		}()
	}

	go func() {
		txHashWG.Wait()
		close(txReceiptCh)
		log.Debugf("SyncedOldBlocks: all tx hashes from block %d:%d have been processed", startBlock, endBlock)
	}()

	finishCh := make(chan struct{})
	go func() {
		processOldTxReceipt(txReceiptCh)
		close(finishCh)
	}()

	for i := startBlock; i < endBlock; i++ {
		blocknumCh <- i
	}
	close(blocknumCh)

	<-finishCh

	log.Debugf("SyncedOldBlocks: all tx receipts from block %d:%d have been processed", startBlock, endBlock)
}

func StartSyncOldBlock(startBlock, endBlock uint64) {
	client, err := blockchain.GetRpcClient()
	if err != nil {
		log.Errorln("SyncedOldBlocks: error getting the eth rpc client")
		log.Errorln(err)
		time.Sleep(time.Second)
		return
	}

	var step uint64 = 100
	for start := startBlock; start < endBlock; start += step {
		end := start + step
		if end > endBlock {
			end = endBlock
		}

		syncOldBlocks(client, start, end, 4)
	}
}
