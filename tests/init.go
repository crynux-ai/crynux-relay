package tests

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"h_relay/api"
	"h_relay/config"
	"os"
)

var Application *gin.Engine = nil

func init() {
	// Initialize api application to serve API test calls

	testAppConfig := &config.AppConfig{}

	testAppConfig.Environment = config.EnvTest
	testAppConfig.Db.Driver = "sqlite"
	testAppConfig.Db.ConnectionString = "data/test_db.sqlite"
	testAppConfig.Log.Level = logrus.DebugLevel.String()
	testAppConfig.Http.Host = "127.0.0.1"
	testAppConfig.Http.Port = "8080"

	err := config.InitDB(testAppConfig)
	if err != nil {
		print(err.Error())
		os.Exit(1)
	}

	Application = api.GetHttpApplication(testAppConfig)
}
