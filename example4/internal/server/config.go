package server

//go:generate ggconfig --interface=Config --output=../gconfig --registry --example=../../configs
type Config interface {
	// Realms returns list of realm configurations
	Realms(defaultValue []RealmInfo) ([]RealmInfo, bool)
	// Host returns server host
	Host(defaultValue string) (string, bool)
	// Port returns server port
	Port(defaultValue int) (int, bool)
}

type RealmInfo struct {
	ID         string   `yaml:"id" json:"id"`
	ClientHost string   `yaml:"clientHost" json:"clientHost"`
	ClientPort int      `yaml:"clientPort" json:"clientPort"`
	Regions    []string `yaml:"regions" json:"regions"`
	Version    string   `yaml:"version" json:"version"`
}
