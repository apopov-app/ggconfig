package main

import (
	"log"
	"os"

	"github.com/apopov-app/ggconfig/example/internal/db"
)

func main() {
	log.Println("🚀 Starting example application...")

	// Пример 1: Использование ENV конфигурации
	log.Println("\n=== ENV Configuration ===")
	envConfig := db.NewConfigDbConfig()
	dbConn, err := db.NewConnection(envConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Printf("✅ Connected with ENV config: %s", dbConn.GetDSN())
	dbConn.Close()

	// Пример 2: Использование YAML конфигурации
	log.Println("\n=== YAML Configuration ===")
	yamlData, err := os.ReadFile("configs/db_example.yaml")
	if err != nil {
		log.Printf("⚠️  YAML config not found, using defaults")
		yamlData = []byte(`db:
  host: "yaml-host"
  port: "5433"
  user: "yaml-user"
  password: "yaml-password"
  name: "yaml-db"
  sslmode: "require"`)
	}

	yamlConfig := db.NewYAMLConfig(yamlData)
	dbConn, err = db.NewConnection(yamlConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Printf("✅ Connected with YAML config: %s", dbConn.GetDSN())
	dbConn.Close()

	// Пример 3: Использование Mock конфигурации для тестов
	log.Println("\n=== Mock Configuration (for tests) ===")
	mockConfig := db.NewMockDbConfig()
	dbConn, err = db.NewConnection(mockConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Printf("✅ Connected with Mock config: %s", dbConn.GetDSN())
	dbConn.Close()

	log.Println("\n🎉 All examples completed successfully!")
}
