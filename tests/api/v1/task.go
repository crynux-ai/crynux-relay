package v1

import (
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"h_relay/api/v1/inference_tasks"
	"h_relay/models"
)

func PrepareAccounts() (addresses []string, privateKeys []string, err error) {

	for i := 0; i < 5; i++ {
		address, pk, err := CreateAccount()
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
		BaseModel:  "sd-v1-5",
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
	task.TaskHash = taskHash

	dataHash, err := task.GetDataHash()
	if err != nil {
		return nil, nil, err
	}
	task.DataHash = dataHash

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
