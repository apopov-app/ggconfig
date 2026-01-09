package main

import (
	"flag"
	"log"

	"github.com/apopov-app/ggconfig/example3/cmd/Bbin/internal/server"
	"github.com/apopov-app/ggconfig/example3/internal/gconfig"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file (optional)")
	flag.Parse()

	log.Println("🚀 Starting Bbin application...")

	// Create GlobalConfig with sources
	global, err := gconfig.NewGlobalConfig(
		gconfig.NewEnvConfig(func(key string) string { return key }),
		gconfig.NewGlobalYamlConfig(*configPath),
	)
	if err != nil {
		log.Fatalf("Failed to create GlobalConfig: %v", err)
	}

	// Get configuration using auto-generated unique name
	serverCfg, ok := global.GetCmdBbinInternalServer()
	if !ok {
		log.Fatal("cmd_Bbin_internal_server config not registered")
	}

	// Package handles defaults internally
	_, addr, err := server.NewFromConfig(serverCfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	log.Printf("✅ Server configured: %s", addr)

	log.Println("\n🎉 Bbin example completed successfully!")
	log.Println("(Server not started in example, but configuration is ready)")
}
