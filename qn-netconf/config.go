package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds the application configuration.
type Config struct {
	SSHPort           int           `json:"ssh_port"`
	ServerBanner      string        `json:"server_banner"`
	FrameEnd          string        `json:"frame_end"`
	ConnectionTimeout time.Duration `json:"connection_timeout_seconds"`
	HostKeyPath       string        `json:"host_key_path"`
	MiyagiSocketPath  string        `json:"miyagi_socket_path"`
	LogLevel          string        `json:"log_level"` // Example: "INFO", "DEBUG"
}

// LoadConfig loads configuration from the given JSON file path.
func LoadConfig(filePath string) (*Config, error) {
	configFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var cfg Config
	err = json.Unmarshal(configFile, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file %s: %w", filePath, err)
	}

	// Convert timeout from seconds to time.Duration
	cfg.ConnectionTimeout = cfg.ConnectionTimeout * time.Second
	return &cfg, nil
}
