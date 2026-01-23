// Package config provides configuration management for GUL components.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Default values
const (
	DefaultSyncInterval    = 1 * time.Minute
	DefaultHTTPTimeout     = 30 * time.Second
	DefaultMaxDownloadSize = 100 * 1024 * 1024 // 100 MB
)

// Config is the main configuration structure for GUL.
type Config struct {
	// Client/SDK configuration
	Client *ClientConfig `json:"client,omitempty"`

	// Publisher configuration
	Publisher *PublisherConfig `json:"publisher,omitempty"`

	// Paths
	DataDir string `json:"data_dir"`
}

// ClientConfig contains configuration for the SDK client.
type ClientConfig struct {
	// ManifestURL is the URL to poll for updates
	ManifestURL string `json:"manifest_url"`

	// PublicKeyHex is the Ed25519 public key in hex format
	PublicKeyHex string `json:"public_key_hex"`

	// StorePath is where to store the local geofence database
	StorePath string `json:"store_path"`

	// SyncInterval is how often to check for updates
	SyncInterval time.Duration `json:"sync_interval"`

	// HTTPTimeout is the timeout for HTTP requests
	HTTPTimeout time.Duration `json:"http_timeout"`

	// MaxDownloadSize is the maximum size of data to download
	MaxDownloadSize int64 `json:"max_download_size"`

	// UserAgent for HTTP requests
	UserAgent string `json:"user_agent"`

	// InsecureSkipVerify disables signature verification (DANGEROUS: for development only!)
	// When true, manifests will be accepted without signature verification.
	// This should NEVER be used in production environments.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

// PublisherConfig contains configuration for the publisher tool.
type PublisherConfig struct {
	// PrivateKeyHex is the Ed25519 private key in hex format
	PrivateKeyHex string `json:"private_key_hex"`

	// KeyID identifies which key to use
	KeyID string `json:"key_id"`

	// OutputDir is where to write generated files
	OutputDir string `json:"output_dir"`

	// CDNBaseURL is the base URL for the CDN
	CDNBaseURL string `json:"cdn_base_url"`

	// CurrentVersion is the starting version number
	CurrentVersion uint64 `json:"current_version"`

	// PreviousDir contains data from previous version (for delta generation)
	PreviousDir string `json:"previous_dir"`
}

// Load loads configuration from a file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Save saves configuration to a file.
func (c *Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Client != nil {
		if err := c.Client.Validate(); err != nil {
			return fmt.Errorf("client config invalid: %w", err)
		}
	}
	if c.Publisher != nil {
		if err := c.Publisher.Validate(); err != nil {
			return fmt.Errorf("publisher config invalid: %w", err)
		}
	}
	return nil
}

// Validate validates the client configuration.
func (c *ClientConfig) Validate() error {
	if c.ManifestURL == "" {
		return fmt.Errorf("manifest_url is required")
	}
	// PublicKeyHex is required unless InsecureSkipVerify is explicitly set
	if c.PublicKeyHex == "" && !c.InsecureSkipVerify {
		return fmt.Errorf("public_key_hex is required (set insecure_skip_verify=true to disable verification, NOT recommended for production)")
	}
	if c.StorePath == "" {
		return fmt.Errorf("store_path is required")
	}
	if c.SyncInterval == 0 {
		c.SyncInterval = DefaultSyncInterval
	}
	if c.HTTPTimeout == 0 {
		c.HTTPTimeout = DefaultHTTPTimeout
	}
	if c.MaxDownloadSize == 0 {
		c.MaxDownloadSize = DefaultMaxDownloadSize
	}
	if c.UserAgent == "" {
		c.UserAgent = "GUL-Client/1.0"
	}
	return nil
}

// Validate validates the publisher configuration.
func (c *PublisherConfig) Validate() error {
	if c.PrivateKeyHex == "" {
		return fmt.Errorf("private_key_hex is required")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output_dir is required")
	}
	if c.CDNBaseURL == "" {
		return fmt.Errorf("cdn_base_url is required")
	}
	if c.CurrentVersion == 0 {
		c.CurrentVersion = 1
	}
	return nil
}

// DefaultClientConfig returns a default client configuration.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		SyncInterval:    DefaultSyncInterval,
		HTTPTimeout:     DefaultHTTPTimeout,
		MaxDownloadSize: DefaultMaxDownloadSize,
		UserAgent:       "GUL-Client/1.0",
	}
}

// DefaultPublisherConfig returns a default publisher configuration.
func DefaultPublisherConfig() *PublisherConfig {
	return &PublisherConfig{
		OutputDir:      "./output",
		CDNBaseURL:     "https://cdn.example.com/geofence",
		CurrentVersion: 1,
	}
}

// ExpandPath expands a path relative to the data directory.
func (c *Config) ExpandPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if c.DataDir != "" {
		return filepath.Join(c.DataDir, path)
	}
	return path
}

// GetClientStorePath returns the full path to the client store.
func (c *Config) GetClientStorePath() string {
	if c.Client != nil && c.Client.StorePath != "" {
		return c.ExpandPath(c.Client.StorePath)
	}
	return c.ExpandPath("geofence.db")
}

// GetPublisherOutputPath returns the full path to the publisher output.
func (c *Config) GetPublisherOutputPath() string {
	if c.Publisher != nil && c.Publisher.OutputDir != "" {
		return c.ExpandPath(c.Publisher.OutputDir)
	}
	return c.ExpandPath("output")
}
