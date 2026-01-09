package server

import (
	"fmt"
	"log"
	"net/http"
)

// Server represents an HTTP server
type Server struct {
	httpServer *http.Server
	addr       string
}

// NewFromConfig creates a new HTTP server using the provided config.
// Defaults are defined here (close to where they're used), not in main.
func NewFromConfig(config Config) (*Server, string, error) {
	if config == nil {
		return nil, "", fmt.Errorf("nil config")
	}

	// Defaults live here, in the package that uses them
	host, _ := config.Host("0.0.0.0")
	port, _ := config.Port(8080)

	addr := fmt.Sprintf("%s:%d", host, port)

	srv := &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: http.NewServeMux(),
		},
		addr: addr,
	}

	log.Printf("[Abin] Server configured: %s", addr)

	return srv, addr, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("[Abin] Starting server on %s", s.addr)
	return s.httpServer.ListenAndServe()
}

// GetAddr returns the server address
func (s *Server) GetAddr() string {
	return s.addr
}
