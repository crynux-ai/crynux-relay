package config

const (
	EnvProduction = "production"
	EnvDebug      = "debug"
	EnvTest       = "test"
)

type AppConfig struct {
	Environment string `mapstructure:"environment"`

	Db struct {
		Driver           string `mapstructure:"driver"`
		ConnectionString string `mapstructure:"connection"`
	} `mapstructure:"db"`

	Log struct {
		Level       string `mapstructure:"level"`
		Output      string `mapstructure:"output"`
		MaxFileSize int    `mapstructure:"max_file_size"`
		MaxDays     int    `mapstructure:"max_days"`
		MaxFileNum  int    `mapstructure:"max_file_num"`
	} `mapstructure:"log"`

	Http struct {
		Host string `mapstructure:"host"`
		Port string `mapstructure:"port"`
	} `mapstructure:"http"`

	DataDir struct {
		InferenceTasks string `mapstructure:"inference_tasks"`
	} `mapstructure:"data_dir"`

	Blockchain struct {
		RpcEndpoint   string `mapstructure:"rpc_endpoint"`
		StartBlockNum uint64 `mapstructure:"start_block_num"`
		GasLimit      uint64 `mapstructure:"gas_limit"`
		Account       struct {
			Address    string `mapstructure:"address"`
			PrivateKey string `mapstructure:"private_key"`
		} `mapstructure:"account"`
		Contracts struct {
			Netstats    string `mapstructure:"netstats"`
			Task        string `mapstructure:"task"`
			Node        string `mapstructure:"node"`
			CrynuxToken string `mapstructure:"crynux_token"`
		} `mapstructure:"contracts"`
	} `mapstructure:"blockchain"`

	TaskSchema struct {
		StableDiffusionInference string `mapstructure:"stable_diffusion_inference"`
		GPTInference             string `mapstructure:"gpt_inference"`
	} `mapstructure:"task_schema"`

	Test struct {
		RootAddress    string `mapstructure:"root_address"`
		RootPrivateKey string `mapstructure:"root_private_key"`
	} `mapstructure:"test"`
}
