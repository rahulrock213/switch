package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
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

// LoadConfig provides a static configuration.
func LoadConfig() (*Config, error) {
	cfg := Config{
		SSHPort:           830,
		ServerBanner:      "SSH-2.0-My NETCONF Server v0.1",
		FrameEnd:          "]]>]]>",
		ConnectionTimeout: 900 * time.Second, // Directly use time.Duration
		HostKeyPath:       "./netconf_host_key",
		MiyagiSocketPath:  "/var/run/miyagi.sock",
		LogLevel:          "INFO",
	}

	// Check if the host key file exists at the hardcoded path.
	// If not, generate it.
	if _, err := os.Stat(cfg.HostKeyPath); os.IsNotExist(err) {
		log.Printf("INFO: Host key file %s not found. Attempting to generate a new key.", cfg.HostKeyPath)

		_, privKey, genErr := ed25519.GenerateKey(rand.Reader)
		if genErr != nil {
			return nil, fmt.Errorf("failed to generate ed25519 key pair: %w", err)
		}

		// Marshal the private key to PKCS#8 ASN.1 DER form.
		privKeyBytes, marshalErr := x509.MarshalPKCS8PrivateKey(privKey)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal private key to PKCS#8: %w", err)
		}

		pemBlock := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privKeyBytes,
		}

		// Create or truncate the key file with 0600 permissions (read/write for owner only)
		keyFile, openErr := os.OpenFile(cfg.HostKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if openErr != nil {
			return nil, fmt.Errorf("failed to open key file %s for writing: %w", cfg.HostKeyPath, err)
		}
		defer keyFile.Close()

		if encodeErr := pem.Encode(keyFile, pemBlock); encodeErr != nil {
			return nil, fmt.Errorf("failed to write PEM data to key file %s: %w", cfg.HostKeyPath, err)
		}
		log.Printf("INFO: Successfully generated and saved new host key to %s", cfg.HostKeyPath)
		// cfg.HostKeyPath is already set to the desired path
	}

	return &cfg, nil
}
