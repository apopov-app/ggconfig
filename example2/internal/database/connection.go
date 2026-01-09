package database

import (
	"fmt"
	"log"
)

// Connection represents a database connection
type Connection struct {
	dsn string
}

// NewFromConfig creates a new database connection using the provided config.
// Defaults are defined here (close to where they're used), not in main.
func NewFromConfig(config Config) (*Connection, error) {
	if config == nil {
		return nil, fmt.Errorf("nil config")
	}

	// Defaults live here, in the package that uses them
	host, _ := config.Host("localhost")
	port, _ := config.Port("5432")
	user, _ := config.User("postgres")
	password, _ := config.Password("password")
	name, _ := config.Name("example")
	sslMode, _ := config.SSLMode("disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, name, sslMode,
	)

	log.Printf("Connecting to database: %s", dsn)

	return &Connection{
		dsn: dsn,
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
