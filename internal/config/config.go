package config

import (
	"github.com/kelseyhightower/envconfig"
)

var (
	config *Configuration
)

type Configuration struct {
	Server struct {
		Host string `envconfig:"SERVER_HOST"`
		Port string `envconfig:"SERVER_PORT"`
	}
	Database struct {
		Address      string `envconfig:"MONGO_ADDRESS"`
		DatabaseName string `envconfig:"MONGO_DATABASE"`
		Collection   string `envconfig:"MONGO_COLLECTION"`
	}
}

func InitConfig() (*Configuration, error) {
	err := envconfig.Process("", config)
	return config, err
}