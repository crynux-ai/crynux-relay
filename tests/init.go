package tests

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"h_relay/api"
	"h_relay/config"
	"h_relay/migrate"
	"os"
)

var Application *gin.Engine = nil

func init() {
	// Initialize api application to serve API test calls

	testAppConfig := &config.AppConfig{}

	testAppConfig.Environment = config.EnvTest
	testAppConfig.Db.Driver = "sqlite"
	testAppConfig.Db.ConnectionString = "/h_relay/data/test_db.sqlite"
	testAppConfig.Log.Level = logrus.DebugLevel.String()
	testAppConfig.Log.Output = "stdout"
	testAppConfig.Http.Host = "127.0.0.1"
	testAppConfig.Http.Port = "8080"

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
