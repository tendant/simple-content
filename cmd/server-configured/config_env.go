package main

import (
	"fmt"

	"github.com/tendant/simple-content/pkg/simplecontent/config"
)

// loadServerConfigFromEnv constructs a ServerConfig by reading process environment variables.
// This keeps environment-specific logic within the executable instead of the library.
func loadServerConfigFromEnv() (*config.ServerConfig, error) {
	cfg, err := config.Load(config.WithEnv(""))
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}
