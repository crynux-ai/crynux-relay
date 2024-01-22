package tests

import (
	"crynux_relay/config"
	"os"
)

func ClearDataFolders() error {
	appConfig := config.GetConfig()
	return removeAllContent(appConfig.DataDir.InferenceTasks)
}

func removeAllContent(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}

	return nil
}
