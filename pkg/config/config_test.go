package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClientConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ClientConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &ClientConfig{
				ManifestURL: "https://example.com/manifest.json",
				PublicKeyHex: "0000000000000000000000000000000000000000000000000000000000000000",
				StorePath:   "/data/geofence.db",
			},
			wantErr: false,
		},
		{
			name: "missing manifest URL",
			cfg: &ClientConfig{
				PublicKeyHex: "0000000000000000000000000000000000000000000000000000000000000000",
				StorePath:   "/data/geofence.db",
			},
			wantErr: true,
		},
		{
			name: "missing public key",
			cfg: &ClientConfig{
				ManifestURL: "https://example.com/manifest.json",
				StorePath:   "/data/geofence.db",
			},
			wantErr: true,
		},
		{
			name: "missing store path",
			cfg: &ClientConfig{
				ManifestURL: "https://example.com/manifest.json",
				PublicKeyHex: "0000000000000000000000000000000000000000000000000000000000000000",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientConfig_DefaultValues(t *testing.T) {
	cfg := &ClientConfig{
		ManifestURL: "https://example.com/manifest.json",
		PublicKeyHex: "0000000000000000000000000000000000000000000000000000000000000000",
		StorePath:   "/data/geofence.db",
	}

	cfg.Validate()

	if cfg.SyncInterval != DefaultSyncInterval {
		t.Errorf("SyncInterval = %v, want %v", cfg.SyncInterval, DefaultSyncInterval)
	}

	if cfg.HTTPTimeout != DefaultHTTPTimeout {
		t.Errorf("HTTPTimeout = %v, want %v", cfg.HTTPTimeout, DefaultHTTPTimeout)
	}

	if cfg.MaxDownloadSize != DefaultMaxDownloadSize {
		t.Errorf("MaxDownloadSize = %d, want %d", cfg.MaxDownloadSize, DefaultMaxDownloadSize)
	}

	if cfg.UserAgent == "" {
		t.Error("UserAgent should have default value")
	}
}

func TestPublisherConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *PublisherConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &PublisherConfig{
				PrivateKeyHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				OutputDir:    "./output",
				CDNBaseURL:   "https://cdn.example.com",
			},
			wantErr: false,
		},
		{
			name: "missing private key",
			cfg: &PublisherConfig{
				OutputDir:  "./output",
				CDNBaseURL: "https://cdn.example.com",
			},
			wantErr: true,
		},
		{
			name: "missing output dir",
			cfg: &PublisherConfig{
				PrivateKeyHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				CDNBaseURL:   "https://cdn.example.com",
			},
			wantErr: true,
		},
		{
			name: "missing CDN base URL",
			cfg: &PublisherConfig{
				PrivateKeyHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				OutputDir:    "./output",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPublisherConfig_DefaultVersion(t *testing.T) {
	cfg := &PublisherConfig{
		PrivateKeyHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		OutputDir:    "./output",
		CDNBaseURL:   "https://cdn.example.com",
	}

	cfg.Validate()

	if cfg.CurrentVersion != 1 {
		t.Errorf("CurrentVersion = %d, want 1", cfg.CurrentVersion)
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := &Config{
		DataDir: tmpDir,
		Client: &ClientConfig{
			ManifestURL: "https://example.com/manifest.json",
			PublicKeyHex: "0000000000000000000000000000000000000000000000000000000000000000",
			StorePath:   "geofence.db",
		},
	}

	// Save
	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Load
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Client.ManifestURL != cfg.Client.ManifestURL {
		t.Errorf("ManifestURL = %s, want %s", loaded.Client.ManifestURL, cfg.Client.ManifestURL)
	}

	if loaded.Client.StorePath != cfg.Client.StorePath {
		t.Errorf("StorePath = %s, want %s", loaded.Client.StorePath, cfg.Client.StorePath)
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid client config", func(t *testing.T) {
		cfg := &Config{
			Client: &ClientConfig{
				ManifestURL: "https://example.com/manifest.json",
				PublicKeyHex: "0000000000000000000000000000000000000000000000000000000000000000",
				StorePath:   "/data/geofence.db",
			},
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate failed: %v", err)
		}
	})

	t.Run("valid publisher config", func(t *testing.T) {
		cfg := &Config{
			Publisher: &PublisherConfig{
				PrivateKeyHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				OutputDir:    "./output",
				CDNBaseURL:   "https://cdn.example.com",
			},
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate failed: %v", err)
		}
	})

	t.Run("invalid client config", func(t *testing.T) {
		cfg := &Config{
			Client: &ClientConfig{
				StorePath: "/data/geofence.db",
			},
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for invalid client config")
		}
	})
}

func TestConfig_ExpandPath(t *testing.T) {
	cfg := &Config{DataDir: "/base"}

	t.Run("absolute path", func(t *testing.T) {
		result := cfg.ExpandPath("/absolute/path")
		if result != "/absolute/path" {
			t.Errorf("ExpandPath = %s, want /absolute/path", result)
		}
	})

	t.Run("relative path", func(t *testing.T) {
		result := cfg.ExpandPath("relative/path")
		if result != "/base/relative/path" {
			t.Errorf("ExpandPath = %s, want /base/relative/path", result)
		}
	})

	t.Run("no data dir", func(t *testing.T) {
		cfg := &Config{}
		result := cfg.ExpandPath("relative/path")
		if result != "relative/path" {
			t.Errorf("ExpandPath = %s, want relative/path", result)
		}
	})
}

func TestConfig_GetClientStorePath(t *testing.T) {
	t.Run("from client config", func(t *testing.T) {
		cfg := &Config{
			DataDir: "/base",
			Client: &ClientConfig{
				ManifestURL: "https://example.com/manifest.json",
				PublicKeyHex: "0000000000000000000000000000000000000000000000000000000000000000",
				StorePath:   "custom.db",
			},
		}
		result := cfg.GetClientStorePath()
		if result != "/base/custom.db" {
			t.Errorf("GetClientStorePath = %s, want /base/custom.db", result)
		}
	})

	t.Run("default path", func(t *testing.T) {
		cfg := &Config{
			DataDir: "/base",
			Client: &ClientConfig{
				ManifestURL: "https://example.com/manifest.json",
				PublicKeyHex: "0000000000000000000000000000000000000000000000000000000000000000",
			},
		}
		result := cfg.GetClientStorePath()
		if result != "/base/geofence.db" {
			t.Errorf("GetClientStorePath = %s, want /base/geofence.db", result)
		}
	})
}

func TestConfig_GetPublisherOutputPath(t *testing.T) {
	t.Run("from publisher config", func(t *testing.T) {
		cfg := &Config{
			DataDir: "/base",
			Publisher: &PublisherConfig{
				PrivateKeyHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				OutputDir:    "custom_output",
				CDNBaseURL:   "https://cdn.example.com",
			},
		}
		result := cfg.GetPublisherOutputPath()
		if result != "/base/custom_output" {
			t.Errorf("GetPublisherOutputPath = %s, want /base/custom_output", result)
		}
	})

	t.Run("default path", func(t *testing.T) {
		cfg := &Config{
			DataDir: "/base",
			Publisher: &PublisherConfig{
				PrivateKeyHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				CDNBaseURL:   "https://cdn.example.com",
			},
		}
		result := cfg.GetPublisherOutputPath()
		if result != "/base/output" {
			t.Errorf("GetPublisherOutputPath = %s, want /base/output", result)
		}
	})
}

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()

	if cfg.SyncInterval != DefaultSyncInterval {
		t.Errorf("SyncInterval = %v, want %v", cfg.SyncInterval, DefaultSyncInterval)
	}

	if cfg.HTTPTimeout != DefaultHTTPTimeout {
		t.Errorf("HTTPTimeout = %v, want %v", cfg.HTTPTimeout, DefaultHTTPTimeout)
	}

	if cfg.MaxDownloadSize != DefaultMaxDownloadSize {
		t.Errorf("MaxDownloadSize = %d, want %d", cfg.MaxDownloadSize, DefaultMaxDownloadSize)
	}
}

func TestDefaultPublisherConfig(t *testing.T) {
	cfg := DefaultPublisherConfig()

	if cfg.OutputDir != "./output" {
		t.Errorf("OutputDir = %s, want ./output", cfg.OutputDir)
	}

	if cfg.CDNBaseURL != "https://cdn.example.com/geofence" {
		t.Errorf("CDNBaseURL = %s, want https://cdn.example.com/geofence", cfg.CDNBaseURL)
	}

	if cfg.CurrentVersion != 1 {
		t.Errorf("CurrentVersion = %d, want 1", cfg.CurrentVersion)
	}
}

func TestClientConfig_UserAgent(t *testing.T) {
	cfg := &ClientConfig{
		ManifestURL: "https://example.com/manifest.json",
		PublicKeyHex: "0000000000000000000000000000000000000000000000000000000000000000",
		StorePath:   "/data/geofence.db",
	}

	cfg.Validate()

	if cfg.UserAgent != "GUL-Client/1.0" {
		t.Errorf("UserAgent = %s, want GUL-Client/1.0", cfg.UserAgent)
	}
}
