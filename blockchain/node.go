package blockchain

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

func GetNodeStatus(ctx context.Context, address common.Address) (uint8, error) {
	nodeContractInstance := GetNodeContractInstance()
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := &bind.CallOpts{
		Pending: false,
		Context: callCtx,
	}

	if err := getLimiter().Wait(callCtx); err != nil {
		return 0, err
	}
	return nodeContractInstance.GetNodeStatus(opts, address)
}
