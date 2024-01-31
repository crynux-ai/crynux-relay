package tests

import (
	"crynux_relay/api/v1/inference_tasks"
	"crynux_relay/blockchain"
	"crynux_relay/config"
	"crynux_relay/models"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const GPTResponseStr = `{
    "model": "gpt2",
    "choices": [
        {
            "finish_reason": "length",
            "message": {
                "role": "assistant",
                "content": "\n\nI have a chat bot, called \"Eleanor\" which was developed by my team on Skype. The only thing I will say is this"
            },
            "index": 0
        }
    ],
    "usage": {
        "prompt_tokens": 11,
        "completion_tokens": 30,
        "total_tokens": 41
    }
}`

func prepareGPTResponseForTask(task *models.InferenceTask) (string, error) {
	appConfig := config.GetConfig()

	resultFolder := filepath.Join(
		appConfig.DataDir.InferenceTasks,
		task.GetTaskIdAsString(),
		"results",
	)

	if err := os.MkdirAll(resultFolder, 0700); err != nil {
		return "", err
	}

	filename := filepath.Join(resultFolder, "0.json")
	if err := os.WriteFile(filename, []byte(GPTResponseStr), 0777); err != nil {
		return "", nil
	}

	resp := inference_tasks.GPTTaskResponse{}
	if err := json.Unmarshal([]byte(GPTResponseStr), &resp); err != nil {
		return "", nil
	}
	h, err := blockchain.GetHashForGPTResponse(resp)
	if err != nil {
		return "", nil
	}

	return hexutil.Encode(h), nil
}
