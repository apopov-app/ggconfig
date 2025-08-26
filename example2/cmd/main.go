package main

import (
	"log"
	"os"

	"github.com/apopov-app/ggconfig/example2/internal/gconfig"
)

func main() {
	yamlData, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read YAML config: %v", err)
	}

	envServer := gconfig.NewServerConfig()
	yamlServer := gconfig.NewServerConfigYAML(yamlData)
	server := gconfig.NewServerConfigAll(envServer, yamlServer)

	host := server.Host("default-host")
	port := server.Port(8080)
	readTimeout := server.ReadTimeout(15)
	writeTimeout := server.WriteTimeout(15)

	log.Printf("Server via composite config → host=%s port=%d readTimeout=%d writeTimeout=%d", host, port, readTimeout, writeTimeout)
}
