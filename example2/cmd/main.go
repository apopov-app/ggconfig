package main

import (
	"log"
	"os"

	"github.com/apopov-app/ggconfig/example2/internal/db"
	"github.com/apopov-app/ggconfig/example2/internal/genconfig"
)

func main() {
	yamlData, err := os.ReadFile("../config.yaml")
	if err != nil {
		log.Fatalf("Failed to read YAML config: %v", err)
	}

	yamlConfig := genconfig.NewYAMLConfig(yamlData)
	dbConn, err := db.NewConnection(yamlConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Printf("✅ Connected with YAML config: %s", dbConn.GetDSN())
	dbConn.Close()
}
