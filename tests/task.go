package tests

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/api/v1/inference_tasks"
	"h_relay/blockchain"
	"h_relay/config"
	"h_relay/models"
	"h_relay/tests/api/v1"
	"os"
	"path"
	"strconv"
)

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

func PrepareControlnetArgs() models.ControlnetArgs {
	return models.ControlnetArgs{
		Preprocess: &models.PreprocessArgs{
			Method: "canny",
			Args: &models.CannyPreprocessArgs{
				LowThreshold:  100,
				HighThreshold: 200,
			},
		},
		ImageDataURL: "image/png,base64:FFFFFF",
		Weight:       100,
		Model:        "stabilityai/sdxl-controlnet-canny",
	}
}

func PrepareTaskConfig() models.TaskConfig {
	return models.TaskConfig{
		ImageWidth:    512,
		ImageHeight:   512,
		NumImages:     9,
		Seed:          51233333,
		Steps:         30,
		SafetyChecker: false,
		CFG:           700,
	}
}

func PrepareTaskArgs() models.TaskArgs {

	controlnetArgs := PrepareControlnetArgs()
	taskConfig := PrepareTaskConfig()

	return models.TaskArgs{
		BaseModel:      "runwayml/stable-diffusion-v1-5",
		Prompt:         "a photo of an old man sitting on a brown chair, by the seaside, with blue sky and white clouds, a dog is lying under his legs, realistic+++, high res+++, masterpiece+++",
		NegativePrompt: "paintings, sketches, worst quality+++++, low quality+++++, normal quality+++++, lowres, normal quality, monochrome++, grayscale++, skin spots, acnes, skin blemishes, age spot, glans, bad hands, bad fingers",
		Controlnet:     &controlnetArgs,
		TaskConfig:     &taskConfig,
		Lora: &models.LoraArgs{
			Model:  "korean-doll-likeness-v2-0",
			Weight: 90,
		},
	}
}

func PrepareRandomTask() *inference_tasks.TaskInput {
	return &inference_tasks.TaskInput{
		TaskId:   999,
		TaskArgs: PrepareTaskArgs(),
	}
}

func PrepareBlockchainConfirmedTask(addresses []string, db *gorm.DB) (*inference_tasks.TaskInput, *models.InferenceTask, error) {

	taskInput := PrepareRandomTask()

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

	dataHash, err := task.GetDataHash()
	if err != nil {
		return nil, nil, err
	}
	task.DataHash = dataHash.Hex()

	task.Prompt = ""
	task.NegativePrompt = ""
	task.BaseModel = ""
	task.TaskConfig = nil
	task.Controlnet = nil
	task.Lora = nil
	task.Refiner = nil
	task.VAE = ""

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
	err = prepareResultImagesForTask(task.GetTaskIdAsString(), taskInput.TaskArgs.TaskConfig.NumImages)
	if err != nil {
		return nil, nil, err
	}

	// Calculate the pHash for the images
	var result []byte

	appConfig := config.GetConfig()
	for i := 0; i < taskInput.TaskArgs.TaskConfig.NumImages; i++ {
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
