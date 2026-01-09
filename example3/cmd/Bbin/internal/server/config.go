package server

//go:generate ggconfig --interface=Config --output=../../../../internal/gconfig --registry
type Config interface {
	// Port returns server port number
	Port(defaultValue int) (int, bool)
	// Host returns server host address
	Host(defaultValue string) (string, bool)
}
