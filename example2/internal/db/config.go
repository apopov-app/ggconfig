package db

import (
	"fmt"
	"log"
)

//go:generate ggconfig --interface=Config --output=internal/genconfig
type Config interface {
	// Host returns database host address
	Host(defaultValue string) string
	// Port returns database port
	Port(defaultValue string) string
	// User returns database user
	User(defaultValue string) string
	// Password returns database password
	Password(defaultValue string) string
	// Name returns database name
	Name(defaultValue string) string
	// SSLMode returns SSL mode
	SSLMode(defaultValue string) string
}

type Connection struct {
	config Config
	dsn    string
}

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

func (c *Connection) GetDSN() string {
	return c.dsn
}

func (c *Connection) Close() {
	log.Println("Closing database connection")
}
