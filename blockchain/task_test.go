package blockchain_test

import (
	"crynux_relay/blockchain"
	"crynux_relay/models"
	"crynux_relay/tests"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func TestGPTRespHash(t *testing.T) {
	resp := models.GPTTaskResponse{}
	if err := json.Unmarshal([]byte(tests.GPTResponseStr), &resp); err != nil {
		t.Error(err)
	}

	hashBytes, err := blockchain.GetHashForGPTResponse(resp)
	if err != nil {
		t.Error(err)
	}
	hash := hexutil.Encode(hashBytes)
	assert.Equal(t, hash, "0x7aa4c9036633f745aa73f03331abb7fa5beb1bc6b7f3688322432a3da61f49c2", "Wrong gpt resh hash")
}
