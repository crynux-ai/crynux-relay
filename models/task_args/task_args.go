package task_args

type TaskConfig struct {
	CFG           int  `json:"cfg" description:"CFG value" validate:"min=1,max=2000" validate:"required"`
	ImageHeight   int  `json:"image_height" description:"Image height" validate:"required,lte=1024"`
	ImageWidth    int  `json:"image_width" description:"Image width" validate:"required,lte=1024"`
	NumImages     int  `json:"num_images" description:"Number of images to generate" validate:"required,min=1,max=9"`
	SafetyChecker bool `json:"safety_checker" description:"Whether to enable the safety checker"`
	Seed          int  `json:"seed" description:"The random seed used to generate images" validate:"required"`
	Steps         int  `json:"steps" description:"Steps" validate:"required,max=100,min=10"`
}

type TaskArgs struct {
	BaseModel      string          `json:"base_model" validate:"required"`
	Controlnet     *ControlnetArgs `json:"controlnet"`
	Lora           *LoraArgs       `json:"lora"`
	NegativePrompt string          `json:"negative_prompt"`
	Prompt         string          `json:"prompt" validate:"required"`
	Refiner        *RefinerArgs    `json:"refiner"`
	TaskConfig     *TaskConfig     `json:"task_config" gorm:"embedded;embeddedPrefix:task_config_" validate:"required"`
	VAE            string          `json:"vae"`
}
