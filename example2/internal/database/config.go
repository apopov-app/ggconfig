package database

//go:generate ggconfig --interface=Config --output=../../internal/gconfig --example=example_configs
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
