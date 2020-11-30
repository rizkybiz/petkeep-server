package main

import (
	"log"

	"github.com/rizkybiz/petkeep-server/api"
	"github.com/rizkybiz/petkeep-server/config"
)

func main() {
	cfg := config.Generate()
	err := api.StartServer(cfg)
	if err != nil {
		log.Fatalf("Could not start the API server: %s", err)
	}
}
