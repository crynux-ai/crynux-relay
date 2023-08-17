package tests

import (
	"github.com/gin-gonic/gin"
	"h_relay/api"
	"h_relay/config"
	"h_relay/migrate"
	"os"
)

var Application *gin.Engine = nil

func init() {

	if err := config.InitConfig(""); err != nil {
		print(err.Error())
		os.Exit(1)
	}

	testAppConfig := config.GetConfig()

	if err := config.InitLog(testAppConfig); err != nil {
		print(err.Error())
		os.Exit(1)
	}

	err := config.InitDB(testAppConfig)
	if err != nil {
		print(err.Error())
		os.Exit(1)
	}

	migrate.InitMigration(config.GetDB())

	if err := migrate.Migrate(); err != nil {
		print(err.Error())
		os.Exit(1)
	}

	Application = api.GetHttpApplication(testAppConfig)
}
