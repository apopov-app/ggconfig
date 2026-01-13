package main

import (
	"flag"
	"log"

	"github.com/apopov-app/ggconfig/example/internal/db"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file (optional)")
	flag.Parse()

	log.Println("🚀 Starting example application...")

	// Main only reads configuration, doesn't define defaults.
	// Defaults are defined in the package that uses them (db package).

	var cfg db.Config

	if *configPath != "" {
		// Option 1: YAML configuration
		log.Println("\n=== Using YAML Configuration ===")
		cfg = db.NewInternalDbConfigYAMLConfig(*configPath)
	} else {
		// Option 2: ENV configuration (default)
		log.Println("\n=== Using ENV Configuration ===")
		cfg = db.NewInternalDbConfigEnvConfig()
	}

	// Package handles defaults internally
	dbConn, err := db.NewFromConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Printf("✅ Connected: %s", dbConn.GetDSN())
	defer dbConn.Close()

	log.Println("\n🎉 Example completed successfully!")
}
