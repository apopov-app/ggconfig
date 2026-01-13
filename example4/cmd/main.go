package main

import (
	"flag"
	"log"

	"github.com/apopov-app/ggconfig/example4/internal/gconfig"
	"github.com/apopov-app/ggconfig/example4/internal/server"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to YAML config file")
	flag.Parse()

	// Create GlobalConfig with YAML and ENV sources
	global, err := gconfig.NewGlobalConfig(
		gconfig.NewEnvConfig(func(key string) string { return key }),
		gconfig.NewGlobalYamlConfig(*configPath),
	)
	if err != nil {
		log.Fatalf("Failed to create global config: %v", err)
	}

	// Get server configuration from registry
	serverCfg, ok := global.GetInternalServer()
	if !ok {
		log.Fatal("server config not registered")
	}

	// Create server from config
	srv, err := server.NewFromConfig(serverCfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Print configuration
	srv.PrintConfig()
}
