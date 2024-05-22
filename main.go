package main

import (
	"crynux_relay/api"
	"crynux_relay/config"
	"crynux_relay/migrate"
	"crynux_relay/tasks"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := config.InitConfig(""); err != nil {
		print("Error reading config file")
		print(err.Error())
		os.Exit(1)
	}

	conf := config.GetConfig()

	if err := config.InitLog(conf); err != nil {
		print("Error initializing log")
		print(err.Error())
		os.Exit(1)
	}

	if err := config.InitDB(conf); err != nil {
		log.Errorln(err.Error())
		os.Exit(1)
	}

	startDBMigration()

	go tasks.StartSyncNetwork()
	go tasks.StartSyncBlock()

	startServer()
}

func startServer() {
	conf := config.GetConfig()

	app := api.GetHttpApplication(conf)
	address := fmt.Sprintf("%s:%s", conf.Http.Host, conf.Http.Port)

	log.Infoln("Starting application server...")

	if err := app.Run(address); err != nil {
		log.Errorln(err.Error())
		os.Exit(1)
	}
}

func startDBMigration() {

	migrate.InitMigration(config.GetDB())

	if err := migrate.Migrate(); err != nil {
		log.Errorln(err.Error())
		os.Exit(1)
	}

	log.Infoln("DB migrations are done!")
}
