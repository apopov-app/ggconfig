package server

//go:generate ggconfig --interface=Config --output=../../internal/gconfig --example=example_configs --alias env.Host=SERVER_ADDRESS_ALIASE
type Config interface {
	// Port returns server port number
	Port(defaultValue int) int
	// Host returns server host address
	Host(defaultValue string) string
	// ReadTimeout returns read timeout in seconds
	ReadTimeout(defaultValue int) int
	// WriteTimeout returns write timeout in seconds
	WriteTimeout(defaultValue int) int
}
