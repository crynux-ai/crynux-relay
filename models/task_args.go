package models

type TaskConfig struct {
	CFG           int  `json:"cfg" description:"CFG value" validate:"min=1,max=2000" validate:"required"`
	ImageHeight   int  `json:"image_height" description:"Image height" validate:"required,lte=1024"`
	ImageWidth    int  `json:"image_width" description:"Image width" validate:"required,lte=1024"`
	NumImages     int  `json:"num_images" description:"Number of images to generate" validate:"required,min=1,max=9"`
	SafetyChecker bool `json:"safety_checker" description:"Whether to enable the safety checker"`
	Seed          int  `json:"seed" description:"The random seed used to generate images" validate:"required"`
	Steps         int  `json:"steps" description:"Steps" validate:"required,max=100,min=10"`
}

type RefinerArgs struct {
	DenoisingCutoff int    `json:"denoising_cutoff" description:"Noise cutoff ratio between base model and refiner" validate:"required,max=100,min=1"`
	Model           string `json:"model" description:"The refiner model name" validate:"required"`
	Steps           int    `json:"steps" description:"Running steps for the refiner" validate:"required,min=10,max=100"`
}

type ControlnetArgs struct {
	Model        string          `json:"model" description:"The controlnet model name" validate:"required"`
	ImageDataURL string          `json:"image_dataurl" description:"The reference image DataURL" validate:"required"`
	Weight       int             `validate:"max=100,min=1" description:"Weight of the controlnet model" validate:"required"`
	Preprocess   *PreprocessArgs `json:"preprocess" description:"Preprocess the image"`
}

type LoraArgs struct {
	Model  string `json:"model" description:"The LoRA model name" validate:"required"`
	Weight int    `json:"weight" description:"The LoRA weight" validate:"required,min=1,max=100"`
}

type TaskArgs struct {
	BaseModel      string          `json:"base_model" validate:"required"`
	Controlnet     *ControlnetArgs `json:"controlnet" gorm:"embedded;embeddedPrefix:controlnet_"`
	Lora           *LoraArgs       `json:"lora" gorm:"embedded;embeddedPrefix:lora_"`
	NegativePrompt string          `json:"negative_prompt"`
	Prompt         string          `json:"prompt" validate:"required"`
	Refiner        *RefinerArgs    `json:"refiner" gorm:"embedded;embeddedPrefix:refiner_"`
	TaskConfig     *TaskConfig     `json:"task_config" gorm:"embedded;embeddedPrefix:task_config_" validate:"required"`
	VAE            string          `json:"vae"`
}
