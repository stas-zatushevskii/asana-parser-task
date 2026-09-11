package main

import (
	"flag"
	"log"

	"asana/internal/app"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to yaml config file")
	flag.Parse()

	application, err := app.New(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
