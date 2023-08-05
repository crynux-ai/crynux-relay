package main

import (
	"fmt"
	"h_relay/api"
	"h_relay/config"
	"os"
)

func main() {

	if err := config.InitConfig(""); err != nil {
		print(err.Error())
		os.Exit(1)
	}

	conf := config.GetConfig()

	err := config.InitDB(conf)
	if err != nil {
		print(err.Error())
		os.Exit(1)
	}

	app := api.GetHttpApplication(conf)
	address := fmt.Sprintf("%s:%s", conf.Http.Host, conf.Http.Port)

	err = app.Run(address)
	if err != nil {
		print(err.Error())
		os.Exit(1)
	}
}
