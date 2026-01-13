package server

import (
	"fmt"
	"log"
)

type Server struct {
	Host   string
	Port   int
	Realms []RealmInfo
}

func NewFromConfig(cfg Config) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	host, _ := cfg.Host("localhost")
	port, _ := cfg.Port(8080)
	realms, ok := cfg.Realms(nil)
	if !ok {
		log.Printf("Warning: No realms configured, using empty list")
		realms = []RealmInfo{}
	}

	return &Server{
		Host:   host,
		Port:   port,
		Realms: realms,
	}, nil
}

func (s *Server) PrintConfig() {
	fmt.Printf("Server Configuration:\n")
	fmt.Printf("  Host: %s\n", s.Host)
	fmt.Printf("  Port: %d\n", s.Port)
	fmt.Printf("  Realms (%d):\n", len(s.Realms))
	for i, realm := range s.Realms {
		fmt.Printf("    [%d] ID: %s\n", i+1, realm.ID)
		fmt.Printf("        Client: %s:%d\n", realm.ClientHost, realm.ClientPort)
		fmt.Printf("        Regions: %v\n", realm.Regions)
		fmt.Printf("        Version: %s\n", realm.Version)
	}
}
