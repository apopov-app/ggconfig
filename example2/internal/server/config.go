package server

//go:generate ggconfig --interface=Config --output=../../internal/gconfig --registry --example=example_configs --alias env.Host=SERVER_ADDRESS_ALIASE
type Config interface {
	// Port returns server port number
	Port(defaultValue int) (int, bool)
	// Host returns server host address
	Host(defaultValue string) (string, bool)
	// ReadTimeout returns read timeout in seconds
	ReadTimeout(defaultValue int) (int, bool)
	// WriteTimeout returns write timeout in seconds
	WriteTimeout(defaultValue int) (int, bool)
}
