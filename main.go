package main

import (
	"fmt"
	"h_relay/api"
	"h_relay/config"
	"h_relay/migrate"
	"os"
)

func main() {

	if len(os.Args) != 2 {
		print("Wrong argument")
		os.Exit(1)
	}

	if err := config.InitConfig(""); err != nil {
		print(err.Error())
		os.Exit(1)
	}

	conf := config.GetConfig()

	if err := config.InitLog(conf); err != nil {
		print(err.Error())
		os.Exit(1)
	}

	if err := config.InitDB(conf); err != nil {
		print(err.Error())
		os.Exit(1)
	}

	cmd := os.Args[1]

	if cmd == "server" {
		startServer()
	} else if cmd == "migration:migrate" {
		startDBMigration()
	} else if cmd == "migration:rollback" {
		startDBRollback()
	} else {
		print("Command not recognized")
		os.Exit(1)
	}
}

func startServer() {
	conf := config.GetConfig()

	app := api.GetHttpApplication(conf)
	address := fmt.Sprintf("%s:%s", conf.Http.Host, conf.Http.Port)

	if err := app.Run(address); err != nil {
		print(err.Error())
		os.Exit(1)
	}
}

func startDBMigration() {

	migrate.InitMigration(config.GetDB())

	if err := migrate.Migrate(); err != nil {
		print(err.Error())
		os.Exit(1)
	}
}

func startDBRollback() {
	if err := migrate.Rollback(); err != nil {
		print(err.Error())
		os.Exit(1)
	}
}
