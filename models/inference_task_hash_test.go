package models_test

import (
	"crynux_relay/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInferenceTaskHash(t *testing.T) {
	taskArgs := `{"base_model": "runwayml/stable-diffusion-v1-5", "prompt": "best quality, ultra high res, photorealistic++++, 1girl, off-shoulder sweater, smiling, faded ash gray messy bun hair+, border light, depth of field, looking at viewer, closeup", "negative_prompt": "paintings, sketches, worst quality+++++, low quality+++++, normal quality+++++, lowres, normal quality, monochrome++, grayscale++, skin spots, acnes, skin blemishes, age spot, glans", "task_config": {"num_images": 1, "safety_checker": false}}`
	taskArgsHash := `0xe8a09714ac70305cbc7c35202d5c77927b4390c0c26274fd33aadd9061d222ce`

	task := &models.InferenceTask{
		TaskArgs: taskArgs,
	}

	generatedHash, err := task.GetTaskHash()
	assert.Nil(t, err, "error get task hash")
	assert.Equal(t, taskArgsHash, generatedHash.Hex())
}
