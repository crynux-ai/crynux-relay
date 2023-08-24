package config

import (
	"github.com/spf13/viper"
)

var appConfig *AppConfig

// InitConfig Init is an exported method that takes the config from the config file
// and unmarshal it into AppConfig struct
func InitConfig(configPath string) error {
	v := viper.New()
	v.SetConfigType("yml")
	v.SetConfigName("config")

	if configPath != "" {
		v.AddConfigPath(configPath)
	} else {
		v.AddConfigPath("/app/config")
		v.AddConfigPath("config")
	}

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	appConfig = &AppConfig{}

	if err := v.Unmarshal(appConfig); err != nil {
		return err
	}

	return nil
}

func GetConfig() *AppConfig {
	return appConfig
}
