package main

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerPort   string `env:"SERVER_PORT" envDefault:"9080"`
	ServerDBPath string `env:"SERVER_DB_PATH" envDefault:"cubik.db"`
}

func LoadConfig() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}
