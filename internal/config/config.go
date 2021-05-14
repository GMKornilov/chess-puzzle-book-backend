package config

import (
	"github.com/kelseyhightower/envconfig"
)

type BackendConfiguration struct {
	Server struct {
		Port string `envconfig:"PORT"`
	}
	Database struct {
		Address      string `envconfig:"MONGO_ADDRESS"`
		DatabaseName string `envconfig:"MONGO_DATABASE"`
		Collection   string `envconfig:"MONGO_COLLECTION"`
	}
	Stockfish struct {
		Path string   `envconfig:"STOCKFISH_PATH"`
		Args []string `envconfig:"STOCKFISH_ARGS"`
	}
}

func InitBackendConfig() (*BackendConfiguration, error) {
	var config BackendConfiguration
	err := envconfig.Process("", &config)
	return &config, err
}

type ScraperConfiguration struct {
	Database struct {
		Address      string `envconfig:"MONGO_ADDRESS"`
		DatabaseName string `envconfig:"MONGO_DATABASE"`
		Collection   string `envconfig:"MONGO_COLLECTION"`
	}
	Stockfish struct {
		Path string   `envconfig:"STOCKFISH_PATH"`
		Args []string `envconfig:"STOCKFISH_ARGS"`
	}
}

func InitScraperConfig() (*ScraperConfiguration, error) {
	var config ScraperConfiguration
	err := envconfig.Process("", &config)
	return &config, err
}
