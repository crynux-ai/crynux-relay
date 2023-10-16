package tests

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/api/v1/inference_tasks"
	"h_relay/blockchain"
	"h_relay/config"
	"h_relay/models"
	v1 "h_relay/tests/api/v1"
	"os"
	"path"
	"strconv"
)

const FullTaskArgsJson string = `{
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

func PrepareRandomTask() (*inference_tasks.TaskInput, error) {
	return &inference_tasks.TaskInput{
		TaskId:   999,
		TaskArgs: FullTaskArgsJson,
	}, nil
}

func PrepareBlockchainConfirmedTask(addresses []string, db *gorm.DB) (*inference_tasks.TaskInput, *models.InferenceTask, error) {

	taskInput, err := PrepareRandomTask()

	if err != nil {
		return nil, nil, err
	}

	task := &models.InferenceTask{
		TaskId:  taskInput.TaskId,
		Creator: addresses[0],
		Status:  models.InferenceTaskCreatedOnChain,
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

func PrepareParamsUploadedTask(addresses []string, db *gorm.DB) (*inference_tasks.TaskInput, *models.InferenceTask, error) {
	taskInput, task, err := PrepareBlockchainConfirmedTask(addresses, db)
	if err != nil {
		return nil, nil, err
	}

	task.TaskArgs = taskInput.TaskArgs
	task.Status = models.InferenceTaskUploaded

	if err := db.Save(task).Error; err != nil {
		return nil, nil, err
	}

	return taskInput, task, nil
}

func PrepareResultUploadedTask(addresses []string, db *gorm.DB) (*inference_tasks.TaskInput, *models.InferenceTask, error) {
	taskInput, task, err := PrepareParamsUploadedTask(addresses, db)
	if err != nil {
		return nil, nil, err
	}

	// Prepare result images
	err = prepareResultImagesForTask(task.GetTaskIdAsString(), 9)
	if err != nil {
		return nil, nil, err
	}

	// Calculate the pHash for the images
	var result []byte

	appConfig := config.GetConfig()
	for i := 0; i < 9; i++ {
		imageFilename := path.Join(
			appConfig.DataDir.InferenceTasks,
			task.GetTaskIdAsString(),
			"results",
			strconv.Itoa(i)+".png",
		)

		imageFile, err := os.Open(imageFilename)
		if err != nil {
			return nil, nil, err
		}

		pHash, err := blockchain.GetPHashForImage(imageFile)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, pHash...)
	}

	resultNode := task.SelectedNodes[1]
	resultNode.IsResultSelected = true
	resultNode.Result = hexutil.Encode(result)

	if err := db.Model(resultNode).Select("IsResultSelected", "Result").Updates(resultNode).Error; err != nil {
		return nil, nil, err
	}

	return taskInput, task, nil
}
