package tests

import (
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/api/v1/inference_tasks"
	"h_relay/models"
	v1 "h_relay/tests/api/v1"
)

const SDTaskArgsJson string = `{
	"base_model": "runwayml/stable-diffusion-v1-5",
	"prompt": "best quality, ultra high res, photorealistic++++, 1girl, off-shoulder sweater, smiling, faded ash gray messy bun hair+, border light, depth of field, looking at viewer, closeup",
	"negative_prompt": "paintings, sketches, worst quality+++++, low quality+++++, normal quality+++++, lowres, normal quality, monochrome++, grayscale++, skin spots, acnes, skin blemishes, age spot, glans",
	"controlnet": {
		"preprocess": {
			"method": "canny",
			"args": {
				"low_threshold": 100,
				"high_threshold": 200
			}
		},
		"model": "lllyasviel/sd-controlnet-canny",
		"weight": 80,
		"image_dataurl": "image/png,base64:FFFFFF"
	},
	"refiner": {
		"model": "stabilityai/stable-diffusion-xl-refiner-1.0",
		"denoising_cutoff": 80,
		"steps": 25
	},
	"lora": {
		"model": "https://civitai.com/api/download/models/34562",
		"weight": 80
	},
	"vae": "stabilityai/sd-vae-ft-mse",
	"textual_inversion": "sd-concepts-library/cat-toy",
	"task_config": {
		"image_width": 512,
		"image_height": 512,
		"num_images": 9,
		"seed": 5123333,
		"steps": 30,
		"safety_checker": false,
		"cfg": 7
	}
}`

const GPTTaskArgsJson string = `{
	"model": "gpt2",
	"messages": [
		{
			"role": "user",
			"content": "I want to create a chat bot. Any suggestions?"
		}
	],
	"generation_config": {
		"max_new_tokens": 30,
		"do_sample": true,
		"num_beams": 1,
		"temperature": 1.0,
		"typical_p": 1.0,
		"top_k": 20,
		"top_p": 1.0,
		"repetition_penalty": 1.0,
		"num_return_sequences": 1
	},
	"seed": 42,
	"dtype": "auto",
	"quantize_bits": 4
}`

var TaskTypes []models.ChainTaskType = []models.ChainTaskType{models.TaskTypeSD, models.TaskTypeLLM}

func PrepareAccounts() (addresses []string, privateKeys []string, err error) {

	for i := 0; i < 5; i++ {
		address, pk, err := v1.CreateAccount()
		if err != nil {
			return nil, nil, err
		}

		addresses = append(addresses, address)
		privateKeys = append(privateKeys, pk)
	}

	log.Debugln(addresses)
	log.Debugln(privateKeys)

	return addresses, privateKeys, nil
}

func PrepareRandomTask(taskType models.ChainTaskType) (*inference_tasks.TaskInput, error) {
	if taskType == models.TaskTypeSD {
		return &inference_tasks.TaskInput{
			TaskId:   999,
			TaskArgs: SDTaskArgsJson,
		}, nil
	} else {
		return &inference_tasks.TaskInput{
			TaskId:   998,
			TaskArgs: GPTTaskArgsJson,
		}, nil
	}
}

func PrepareBlockchainConfirmedTask(taskType models.ChainTaskType, addresses []string, db *gorm.DB) (*inference_tasks.TaskInput, *models.InferenceTask, error) {

	taskInput, err := PrepareRandomTask(taskType)

	if err != nil {
		return nil, nil, err
	}

	task := &models.InferenceTask{
		TaskId:    taskInput.TaskId,
		Creator:   addresses[0],
		Status:    models.InferenceTaskCreatedOnChain,
		TaskType:  taskType,
		VramLimit: 8,
	}

	task.TaskArgs = taskInput.TaskArgs

	taskHash, err := task.GetTaskHash()
	if err != nil {
		return nil, nil, err
	}
	task.TaskHash = taskHash.Hex()

	task.TaskArgs = ""

	if err := db.Create(task).Error; err != nil {
		return nil, nil, err
	}

	for i := 0; i < 3; i++ {
		association := db.Model(task).Association("SelectedNodes")
		if err := association.Append(&models.SelectedNode{NodeAddress: addresses[1+i]}); err != nil {
			return nil, nil, err
		}
	}

	return taskInput, task, nil
}

func PrepareParamsUploadedTask(taskType models.ChainTaskType, addresses []string, db *gorm.DB) (*inference_tasks.TaskInput, *models.InferenceTask, error) {
	taskInput, task, err := PrepareBlockchainConfirmedTask(taskType, addresses, db)
	if err != nil {
		return nil, nil, err
	}

	task.TaskArgs = taskInput.TaskArgs
	task.Status = models.InferenceTaskParamsUploaded

	if err := db.Save(task).Error; err != nil {
		return nil, nil, err
	}

	return taskInput, task, nil
}

func PreparePendingResultsTask(taskType models.ChainTaskType, addresses []string, db *gorm.DB) (*inference_tasks.TaskInput, *models.InferenceTask, error) {
	taskInput, task, err := PrepareBlockchainConfirmedTask(taskType, addresses, db)
	if err != nil {
		return nil, nil, err
	}

	resultNode := task.SelectedNodes[1]
	resultNode.IsResultSelected = true
	if taskType == models.TaskTypeSD {
		pHash, err := prepareResultImagesForTask(task, 9)
		if err != nil {
			return nil, nil, err
		}

		resultNode.Result = pHash
	} else {
		h, err := prepareGPTResponseForTask(task)
		if err != nil {
			return nil, nil, err
		}

		resultNode.Result = h
	}

	if err := db.Model(resultNode).Select("IsResultSelected", "Result").Updates(resultNode).Error; err != nil {
		return nil, nil, err
	}

	task.TaskArgs = taskInput.TaskArgs
	task.Status = models.InferenceTaskPendingResults

	if err := db.Save(task).Error; err != nil {
		return nil, nil, err
	}

	return taskInput, task, nil
}

func PrepareResultUploadedTask(taskType models.ChainTaskType, addresses []string, db *gorm.DB) (*inference_tasks.TaskInput, *models.InferenceTask, error) {
	taskInput, task, err := PreparePendingResultsTask(taskType, addresses, db)
	if err != nil {
		return nil, nil, err
	}

	task.Status = models.InferenceTaskResultsUploaded

	if err := db.Save(task).Error; err != nil {
		return nil, nil, err
	}

	return taskInput, task, nil
}
