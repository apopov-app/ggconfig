package db

import (
	"fmt"
	"log"
)

//go:generate ggconfig --interface=Config --example=configs
type Config interface {
	// Host returns database host address
	Host(defaultValue string) string
	// Port returns database port number
	Port(defaultValue string) string
	// User returns database username
	User(defaultValue string) string
	// Password returns database password
	Password(defaultValue string) string
	// Name returns database name
	Name(defaultValue string) string
	// SSLMode returns SSL mode configuration
	SSLMode(defaultValue string) string
}

// Connection represents a database connection
type Connection struct {
	config Config
	dsn    string
}

// NewConnection creates a new database connection using the provided config
func NewConnection(config Config) (*Connection, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host("localhost"),
		config.Port("5432"),
		config.User("postgres"),
		config.Password("password"),
		config.Name("example"),
		config.SSLMode("disable"),
	)

	log.Printf("Connecting to database: %s", dsn)

	return &Connection{
		config: config,
		dsn:    dsn,
	}, nil
}

// Close closes the database connection
func (c *Connection) Close() error {
	log.Println("Closing database connection")
	return nil
}

// GetDSN returns the connection string
func (c *Connection) GetDSN() string {
	return c.dsn
}
