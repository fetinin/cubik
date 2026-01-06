package main

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds application configuration from environment variables
type Config struct {
	// ServerPort is the HTTP server port
	ServerPort string `env:"SERVER_PORT" envDefault:"9080"`

	// ServerDBPath is the SQLite database file path
	ServerDBPath string `env:"SERVER_DB_PATH" envDefault:"cubik.db"`
}

// LoadConfig reads configuration from environment variables
// Returns populated Config struct or error
func LoadConfig() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}
