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

func PreparePoseConfig() models.PoseConfig {
	return models.PoseConfig{
		Preprocess: false,
		DataURL:    "",
		PoseWeight: 1,
	}
}

func PrepareTaskConfig() models.TaskConfig {
	return models.TaskConfig{
		ImageWidth:  512,
		ImageHeight: 512,
		LoraWeight:  1,
		NumImages:   9,
		Seed:        51233333,
		Steps:       40,
	}
}

func PrepareRandomTask() *inference_tasks.TaskInput {
	return &inference_tasks.TaskInput{
		TaskId:     999,
		BaseModel:  "stable-diffusion-v1-5-pruned",
		LoraModel:  "",
		Prompt:     "a silly man sitting on a brown chair",
		Pose:       PreparePoseConfig(),
		TaskConfig: PrepareTaskConfig(),
	}
}

func PrepareBlockchainConfirmedTask(addresses []string, db *gorm.DB) (*inference_tasks.TaskInput, *models.InferenceTask, error) {

	taskInput := PrepareRandomTask()

	task := &models.InferenceTask{
		TaskId:     taskInput.TaskId,
		Creator:    addresses[0],
		Prompt:     taskInput.Prompt,
		BaseModel:  taskInput.BaseModel,
		LoraModel:  taskInput.LoraModel,
		TaskConfig: &taskInput.TaskConfig,
		Pose:       &taskInput.Pose,
		Status:     models.InferenceTaskCreatedOnChain,
	}

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
	task.BaseModel = ""
	task.LoraModel = ""
	task.TaskConfig = nil
	task.Pose = nil

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

	task.Prompt = taskInput.Prompt
	task.BaseModel = taskInput.BaseModel
	task.LoraModel = taskInput.LoraModel
	task.TaskConfig = &taskInput.TaskConfig
	task.Pose = &taskInput.Pose
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
	err = prepareResultImagesForTask(task.GetTaskIdAsString(), taskInput.TaskConfig.NumImages)
	if err != nil {
		return nil, nil, err
	}

	// Calculate the pHash for the images
	var result []byte

	appConfig := config.GetConfig()
	for i := 0; i < taskInput.TaskConfig.NumImages; i++ {
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
