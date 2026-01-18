// Package main implements the GUL publisher CLI tool.
// This tool is used by administrators to create and publish geofence updates.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/iannil/geofence-updater-lite/internal/version"
	"github.com/iannil/geofence-updater-lite/pkg/config"
	"github.com/iannil/geofence-updater-lite/pkg/crypto"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
	"github.com/iannil/geofence-updater-lite/pkg/publisher"
)

var (
	showVersion = flag.Bool("version", false, "show version information")
	configFile  = flag.String("config", "", "path to config file")
	outputDir   = flag.String("output", "./output", "output directory for generated files")
	dbPath      = flag.String("db", "./geofence.db", "path to fence database")
	keyFile     = flag.String("key", "", "path to private key file (hex encoded)")
	keyID       = flag.String("key-id", "", "key identifier")
	cdnBase     = flag.String("cdn", "", "CDN base URL")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("GUL Publisher %s\n", version.String())
		os.Exit(0)
	}

	// Execute command
	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	cmd := args[0]

	// Handle commands that don't need config
	if cmd == "keys" {
		runKeys()
		return
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("GUL Publisher %s starting...", version.String())

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Handle commands
	switch cmd {
	case "init":
		runInit(cfg)
	case "add":
		if len(args) < 2 {
			log.Fatal("Usage: add <fence.json>")
		}
		runAdd(cfg, args[1])
	case "publish":
		runPublish(cfg)
	case "list":
		runList(cfg)
	case "remove":
		if len(args) < 2 {
			log.Fatal("Usage: remove <fence-id>")
		}
		runRemove(cfg, args[1])
	default:
		log.Fatalf("Unknown command: %s", cmd)
	}
}

func loadConfig() (*config.PublisherConfig, error) {
	cfg := config.DefaultPublisherConfig()

	if *configFile != "" {
		mainCfg, err := config.Load(*configFile)
		if err != nil {
			return nil, err
		}
		if mainCfg.Publisher != nil {
			cfg = mainCfg.Publisher
		}
	}

	// Override with CLI flags
	if *outputDir != "./output" {
		cfg.OutputDir = *outputDir
	}
	if *keyFile != "" {
		cfg.PrivateKeyHex = *keyFile
	}
	if *keyID != "" {
		cfg.KeyID = *keyID
	}
	if *cdnBase != "" {
		cfg.CDNBaseURL = *cdnBase
	}

	// If no private key provided, try to read from file
	if cfg.PrivateKeyHex == "" {
		if keyData, err := os.ReadFile("private.key"); err == nil {
			cfg.PrivateKeyHex = string(keyData)
		}
	}

	return cfg, cfg.Validate()
}

// getStorePath returns the database path.
func getStorePath() string {
	if *dbPath != "./geofence.db" {
		return *dbPath
	}
	if *outputDir != "./output" {
		return filepath.Join(*outputDir, "geofence.db")
	}
	return "./geofence.db"
}

func printUsage() {
	fmt.Println("GUL Publisher - Geofence publishing tool")
	fmt.Println("\nCommands:")
	fmt.Println("  init        Initialize a new geofence database")
	fmt.Println("  add         Add a new fence to the database")
	fmt.Println("  remove      Remove a fence from the database")
	fmt.Println("  list        List all fences in the database")
	fmt.Println("  publish     Publish an update to the CDN")
	fmt.Println("  keys        Generate a new key pair")
	fmt.Println("\nFlags:")
	flag.PrintDefaults()
}

func runInit(cfg *config.PublisherConfig) {
	storePath := getStorePath()
	log.Printf("Initializing new geofence database at %s...", storePath)

	ctx := context.Background()
	if err := publisher.Initialize(ctx, cfg); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	log.Printf("Initialized database at %s", storePath)
}

func runAdd(cfg *config.PublisherConfig, fenceFile string) {
	log.Printf("Adding fence from %s...", fenceFile)

	ctx := context.Background()
	pub, err := publisher.NewPublisher(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	// Read fence JSON
	data, err := os.ReadFile(fenceFile)
	if err != nil {
		log.Fatalf("Failed to read fence file: %v", err)
	}

	var fence geofence.FenceItem
	if err := json.Unmarshal(data, &fence); err != nil {
		log.Fatalf("Failed to parse fence: %v", err)
	}

	// Sign and add fence
	if err := pub.SignAndAdd(ctx, &fence); err != nil {
		log.Fatalf("Failed to add fence: %v", err)
	}

	log.Printf("Added fence %s (type=%s, priority=%d)", fence.ID, fence.Type, fence.Priority)
}

func runRemove(cfg *config.PublisherConfig, fenceID string) {
	log.Printf("Removing fence %s...", fenceID)

	ctx := context.Background()
	pub, err := publisher.NewPublisher(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	if err := pub.DeleteFence(ctx, fenceID); err != nil {
		log.Fatalf("Failed to remove fence: %v", err)
	}

	log.Printf("Removed fence %s", fenceID)
}

func runList(cfg *config.PublisherConfig) {
	ctx := context.Background()
	pub, err := publisher.NewPublisher(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	fences, err := pub.ListFences(ctx)
	if err != nil {
		log.Fatalf("Failed to list fences: %v", err)
	}

	currentVer, _ := pub.GetCurrentVersion(ctx)

	fmt.Printf("\n=== Geofence Database (v%d) ===\n", currentVer)
	fmt.Printf("Total fences: %d\n\n", len(fences))

	for _, f := range fences {
		activeStatus := "Inactive"
		if f.IsActiveNow() {
			activeStatus = "Active"
		}
		fmt.Printf("  %s: %s (type=%s, priority=%d, %s)\n",
			f.ID, f.Name, f.Type, f.Priority, activeStatus)
	}
	fmt.Println("================================")
}

func runPublish(cfg *config.PublisherConfig) {
	log.Printf("Publishing update to %s...", cfg.CDNBaseURL)

	ctx := context.Background()
	pub, err := publisher.NewPublisher(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	// Get all fences from database
	fences, err := pub.ListFences(ctx)
	if err != nil {
		log.Fatalf("Failed to get fences: %v", err)
	}

	// Convert to value slice
	fenceValues := make([]geofence.FenceItem, len(fences))
	for i, f := range fences {
		fenceValues[i] = *f
	}

	// Publish new version
	result, err := pub.Publish(ctx, fenceValues)
	if err != nil {
		log.Fatalf("Failed to publish: %v", err)
	}

	log.Printf("Publish complete!")
	log.Printf("  Version: %d -> %d", result.PreviousVersion, result.Version)
	log.Printf("  Fences: %d", result.FencesCount)
	log.Printf("  Snapshot: %s (%d bytes)", filepath.Base(result.SnapshotPath), result.SnapshotSize)
	if result.DeltaPath != "" {
		log.Printf("  Delta: %s (%d bytes)", filepath.Base(result.DeltaPath), result.DeltaSize)
	}
	log.Printf("  Manifest: %s", result.ManifestPath)
}

func runKeys() {
	log.Println("Generating new Ed25519 key pair...")

	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	fmt.Println("\nGenerated key pair:")
	fmt.Printf("Key ID: %s\n", keyPair.KeyID)
	fmt.Printf("Public Key: %s\n", crypto.MarshalPublicKeyHex(keyPair.PublicKey))
	fmt.Printf("Private Key: %s\n", crypto.MarshalPrivateKeyHex(keyPair.PrivateKey))

	fmt.Println("\nIMPORTANT: Store the private key securely!")
	fmt.Println("Never share the private key.")
	fmt.Println("\nTo save keys to files:")
	fmt.Printf("  echo '%s' > public.key\n", crypto.MarshalPublicKeyHex(keyPair.PublicKey))
	fmt.Printf("  echo '%s' > private.key\n", crypto.MarshalPrivateKeyHex(keyPair.PrivateKey))
}
