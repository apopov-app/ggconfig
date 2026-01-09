package db

//go:generate ggconfig --interface=Config --example=configs
type Config interface {
	// Host returns database host address
	Host(defaultValue string) (string, bool)
	// Port returns database port number
	Port(defaultValue string) (string, bool)
	// User returns database username
	User(defaultValue string) (string, bool)
	// Password returns database password
	Password(defaultValue string) (string, bool)
	// Name returns database name
	Name(defaultValue string) (string, bool)
	// SSLMode returns SSL mode configuration
	SSLMode(defaultValue string) (string, bool)
}
