package main

import (
	"sourcecrawler/app"
	"sourcecrawler/config"
)

func main() {

	config := config.GetConfig()

	app := &app.App{}
	app.Initialize(config)
	app.Run(":3000")

}
