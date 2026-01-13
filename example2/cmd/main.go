package main

import (
	"flag"
	"log"

	"github.com/apopov-app/ggconfig/example2/internal/database"
	"github.com/apopov-app/ggconfig/example2/internal/gconfig"
	"github.com/apopov-app/ggconfig/example2/internal/server"
)

func main() {
	configPath := flag.String("config", "", "path to YAML config file (optional)")
	flag.Parse()

	log.Println("🚀 Starting example2 application (GlobalConfig)...")

	// Main only reads configuration, doesn't define defaults.
	// Defaults are defined in the packages that use them (database, server packages).

	// Create GlobalConfig with sources (order matters: ENV → YAML → default)
	// NewGlobalConfig, NewEnvConfig, NewGlobalYamlConfig are generated in gconfig package
	global, err := gconfig.NewGlobalConfig(
		gconfig.NewEnvConfig(func(key string) string { return key }),
		gconfig.NewGlobalYamlConfig(*configPath),
	)
	if err != nil {
		log.Fatalf("Failed to create GlobalConfig: %v", err)
	}

	// Get configurations from registry
	// Each package registered with --registry gets a Get<Pkg>() method on GlobalConfig
	dbCfg, ok := global.GetInternalDatabase()
	if !ok {
		log.Fatal("database config not registered")
	}

	serverCfg, ok := global.GetInternalServer()
	if !ok {
		log.Fatal("server config not registered")
	}

	// Packages handle defaults internally
	dbConn, err := database.NewFromConfig(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Printf("✅ Database connected: %s", dbConn.GetDSN())
	defer dbConn.Close()

	_, addr, err := server.NewFromConfig(serverCfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	log.Printf("✅ Server configured: %s", addr)

	log.Println("\n🎉 Example2 completed successfully!")
	log.Println("(Server not started in example, but configuration is ready)")
}
