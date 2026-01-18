// Package main provides an example of using the GUL SDK.
// This demonstrates how a drone or ground control station would
// integrate with the geofence update system.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iannil/geofence-updater-lite/internal/version"
	"github.com/iannil/geofence-updater-lite/pkg/config"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
	"github.com/iannil/geofence-updater-lite/pkg/sync"
)

var (
	showVersion = flag.Bool("version", false, "show version information")
	manifestURL = flag.String("manifest", "", "URL to manifest file or CDN base URL")
	publicKey   = flag.String("public-key", "", "Ed25519 public key (hex)")
	storePath   = flag.String("store", "./geofence.db", "path to local store")
	interval    = flag.Duration("interval", time.Minute, "sync interval")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("GUL SDK Example %s\n", version.String())
		os.Exit(0)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("GUL SDK Example %s starting...", version.String())

	// Load configuration
	cfg := loadConfig()

	// Create syncer
	ctx := context.Background()
	syncer, err := sync.NewSyncer(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create syncer: %v", err)
	}
	defer syncer.Close()

	// Start auto-sync in background
	results := syncer.StartAutoSync(ctx, *interval)

	log.Printf("Started geofence updater, syncing every %v", *interval)

	// Handle sync results in background
	go func() {
		for result := range results {
			if result.Error != nil {
				log.Printf("[Sync] Error: %v", result.Error)
				continue
			}
			if result.UpToDate {
				log.Printf("[Sync] Already up to date (v%d)", result.CurrentVer)
			} else {
				log.Printf("[Sync] Updated: v%d -> v%d in %v",
					result.PreviousVer, result.CurrentVer, result.Duration)
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}

func loadConfig() *config.ClientConfig {
	cfg := &config.ClientConfig{
		ManifestURL:    *manifestURL,
		StorePath:      *storePath,
		SyncInterval:   *interval,
		HTTPTimeout:    config.DefaultHTTPTimeout,
		MaxDownloadSize: config.DefaultMaxDownloadSize,
		UserAgent:      "GUL-SDK-Example/1.0",
	}

	// Load public key
	if *publicKey != "" {
		cfg.PublicKeyHex = *publicKey
	} else {
		// Use a test key for demo (DO NOT USE IN PRODUCTION)
		log.Println("WARNING: Using test public key - DO NOT USE IN PRODUCTION")
		cfg.PublicKeyHex = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	return cfg
}

// Updater is the main SDK client for geofence updates (legacy API).
type Updater struct {
	syncer *sync.Syncer
}

// NewUpdater creates a new geofence updater.
func NewUpdater(cfg *config.ClientConfig) (*Updater, error) {
	ctx := context.Background()
	s, err := sync.NewSyncer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Updater{syncer: s}, nil
}

// StartAutoSync starts the automatic sync loop (legacy API).
func (u *Updater) StartAutoSync(stop chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stop
		cancel()
	}()

	results := u.syncer.StartAutoSync(ctx, time.Minute)
	for range results {
		// Results handled in main loop
	}
}

// Check checks if a location is allowed for flight.
func (u *Updater) Check(lat, lon float64) (bool, *geofence.FenceItem) {
	ctx := context.Background()
	allowed, restriction, err := u.syncer.Check(ctx, lat, lon)
	if err != nil {
		log.Printf("Check error: %v", err)
		return true, nil
	}
	return allowed, restriction
}

// LoadFences loads fences from a JSON file (for testing).
func (u *Updater) LoadFences(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var collection geofence.FenceCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return err
	}

	log.Printf("Loaded %d fences from %s", len(collection.Items), path)
	return nil
}

// PrintStatus prints the current updater status.
func (u *Updater) PrintStatus() {
	fmt.Printf("\n=== GUL Updater Status ===\n")
	fmt.Printf("Version: %d\n", u.syncer.GetCurrentVersion())
	fmt.Printf("Last Check: %v\n", u.syncer.GetLastCheckTime())
	fmt.Printf("Last Sync: %v\n", u.syncer.GetLastSyncTime())
	fmt.Printf("=========================\n")
}

// PrintMatchingFences prints all fences that match a location.
func (u *Updater) PrintMatchingFences(lat, lon float64) {
	ctx := context.Background()

	fmt.Printf("\n=== Fence Check for (%.6f, %.6f) ===\n", lat, lon)

	allowed, restriction, err := u.syncer.Check(ctx, lat, lon)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if allowed {
		fmt.Println("Status: ALLOWED")
	} else {
		fmt.Println("Status: NOT ALLOWED")
		if restriction != nil {
			fmt.Printf("Reason: %s - %s\n", restriction.Name, restriction.Description)
		}
	}

	fmt.Println("================================")
}
