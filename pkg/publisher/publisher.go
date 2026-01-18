// Package publisher provides functionality for publishing geofence updates
// to a CDN or static file server.
package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/binarydiff"
	"github.com/iannil/geofence-updater-lite/pkg/config"
	"github.com/iannil/geofence-updater-lite/pkg/crypto"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
	"github.com/iannil/geofence-updater-lite/pkg/merkle"
	"github.com/iannil/geofence-updater-lite/pkg/storage"
)

// Publisher handles publishing of geofence updates.
type Publisher struct {
	store      *storage.SQLiteStore
	cfg        *config.PublisherConfig
	keyPair    *crypto.KeyPair
	currentVer uint64
}

// NewPublisher creates a new geofence publisher.
func NewPublisher(ctx context.Context, cfg *config.PublisherConfig) (*Publisher, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Derive key pair from private key
	privateKey, err := crypto.UnmarshalPrivateKeyHex(cfg.PrivateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	keyPair, err := crypto.DeriveKeyPair(nil, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key pair: %w", err)
	}

	// Override key ID if provided
	if cfg.KeyID != "" {
		keyPair.KeyID = cfg.KeyID
	}

	// Ensure output directory exists
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine store path (use default if not specified)
	storePath := "./geofence.db"
	if cfg.OutputDir != "" {
		storePath = filepath.Join(cfg.OutputDir, "geofence.db")
	}

	// Open storage
	store, err := storage.Open(ctx, &storage.Config{Path: storePath})
	if err != nil {
		return nil, fmt.Errorf("failed to open storage: %w", err)
	}

	// Load current version
	currentVer, err := store.GetVersion(ctx)
	if err != nil {
		currentVer = 0 // First time
	}

	return &Publisher{
		store:      store,
		cfg:        cfg,
		keyPair:    keyPair,
		currentVer: currentVer,
	}, nil
}

// PublishResult contains the result of a publish operation.
type PublishResult struct {
	Version         uint64
	ManifestPath    string
	SnapshotPath    string
	DeltaPath       string
	PreviousVersion uint64
	FencesCount     int
	DeltaSize       int64
	SnapshotSize    int64
	PublishTime     time.Time
}

// Publish creates and publishes a new version with the given fences.
func (p *Publisher) Publish(ctx context.Context, fences []geofence.FenceItem) (*PublishResult, error) {
	startTime := time.Now()

	// Get old fences for delta calculation
	oldFences, _ := p.getCurrentFences(ctx)

	// Increment version
	newVersion := p.currentVer + 1

	// Sign each fence
	for i := range fences {
		if err := p.signFence(&fences[i]); err != nil {
			return nil, fmt.Errorf("failed to sign fence %s: %w", fences[i].ID, err)
		}
	}

	// Build Merkle tree
	tree, err := merkle.NewTree(fences)
	if err != nil {
		return nil, fmt.Errorf("failed to build Merkle tree: %w", err)
	}

	// Compute root hash
	rootHash := tree.RootHash()

	// Create snapshot
	snapshotData, snapshotSize, err := merkle.CreateSnapshot(fences)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Compute snapshot hash
	snapshotHash := crypto.ComputeSHA256(snapshotData)

	// Create delta if there's a previous version
	var deltaData []byte
	var deltaSize int64
	var deltaPath string
	var deltaHash []byte

	if newVersion > 1 && len(oldFences) > 0 {
		delta, err := binarydiff.Diff(oldFences, fences)
		if err == nil {
			delta.FromVersion = p.currentVer
			delta.ToVersion = newVersion
			deltaData = delta.DiffData
			deltaSize = int64(len(deltaData))
			deltaHash = delta.DiffHash
			deltaPath = fmt.Sprintf("/patches/v%d_to_v%d.bin", p.currentVer, newVersion)
		}
	}

	// Create manifest
	manifest := &geofence.Manifest{
		Version:      newVersion,
		Timestamp:    time.Now().Unix(),
		RootHash:     rootHash[:],
		SnapshotURL:  fmt.Sprintf("/snapshots/v%d.bin", newVersion),
		SnapshotSize: uint64(snapshotSize),
		SnapshotHash: snapshotHash,
		Message:      fmt.Sprintf("Version %d - %d fences", newVersion, len(fences)),
	}

	if deltaPath != "" {
		manifest.DeltaURL = deltaPath
		manifest.DeltaSize = uint64(deltaSize)
		manifest.DeltaHash = deltaHash
	}

	// Sign manifest
	manifestData, _ := manifest.MarshalBinaryForSigning()
	manifest.SetSignature(p.keyPair.Sign(manifestData), p.keyPair.KeyID)

	// Write files
	snapshotPath := filepath.Join(p.cfg.OutputDir, fmt.Sprintf("v%d.bin", newVersion))
	if err := os.WriteFile(snapshotPath, snapshotData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write snapshot: %w", err)
	}

	manifestPath := filepath.Join(p.cfg.OutputDir, "manifest.json")
	if err := p.writeManifest(manifest, manifestPath); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	// Write delta if exists
	if len(deltaData) > 0 {
		deltaFullPath := filepath.Join(p.cfg.OutputDir, deltaPath[1:]) // Remove leading /
		deltaDir := filepath.Dir(deltaFullPath)
		if err := os.MkdirAll(deltaDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create delta directory: %w", err)
		}
		if err := os.WriteFile(deltaFullPath, deltaData, 0644); err != nil {
			return nil, fmt.Errorf("failed to write delta: %w", err)
		}
	}

	// Update storage with new fences and manifest
	if err := p.updateStorage(ctx, fences, manifest); err != nil {
		return nil, fmt.Errorf("failed to update storage: %w", err)
	}

	// Update current version
	p.currentVer = newVersion

	return &PublishResult{
		Version:         newVersion,
		ManifestPath:    manifestPath,
		SnapshotPath:    snapshotPath,
		DeltaPath:       deltaPath,
		PreviousVersion: p.currentVer - 1,
		FencesCount:     len(fences),
		DeltaSize:       deltaSize,
		SnapshotSize:    snapshotSize,
		PublishTime:     startTime,
	}, nil
}

// SignAndAdd signs and adds a single fence to the database.
func (p *Publisher) SignAndAdd(ctx context.Context, fence *geofence.FenceItem) error {
	// Sign the fence
	if err := p.signFence(fence); err != nil {
		return err
	}

	// Add to storage
	if err := p.store.AddFence(ctx, fence); err != nil {
		return fmt.Errorf("failed to add fence to storage: %w", err)
	}

	return nil
}

// SignAndUpdate signs and updates a fence in the database.
func (p *Publisher) SignAndUpdate(ctx context.Context, fence *geofence.FenceItem) error {
	// Sign the fence
	if err := p.signFence(fence); err != nil {
		return err
	}

	// Update in storage
	if err := p.store.UpdateFence(ctx, fence); err != nil {
		return fmt.Errorf("failed to update fence in storage: %w", err)
	}

	return nil
}

// signFence signs a fence item with the publisher's key.
func (p *Publisher) signFence(fence *geofence.FenceItem) error {
	// Create fence data for signing (without signature field)
	fenceData, err := json.Marshal(struct {
		ID          string
		Type        geofence.FenceType
		Geometry    geofence.Geometry
		StartTS     int64
		EndTS       int64
		Priority    uint32
		MaxAltitude uint32
		MaxSpeed    uint32
		Name        string
		Description string
	}{
		ID:          fence.ID,
		Type:        fence.Type,
		Geometry:    fence.Geometry,
		StartTS:     fence.StartTS,
		EndTS:       fence.EndTS,
		Priority:    fence.Priority,
		MaxAltitude: fence.MaxAltitude,
		MaxSpeed:    fence.MaxSpeed,
		Name:        fence.Name,
		Description: fence.Description,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal fence: %w", err)
	}

	// Sign the fence data
	fence.Signature = p.keyPair.Sign(fenceData)
	fence.KeyID = p.keyPair.KeyID

	return nil
}

// getCurrentFences retrieves all current fences from storage.
func (p *Publisher) getCurrentFences(ctx context.Context) ([]geofence.FenceItem, error) {
	fencePtrs, err := p.store.ListFences(ctx)
	if err != nil {
		return nil, err
	}

	fences := make([]geofence.FenceItem, len(fencePtrs))
	for i, f := range fencePtrs {
		fences[i] = *f
	}
	return fences, nil
}

// updateStorage updates the storage with new fences and manifest.
func (p *Publisher) updateStorage(ctx context.Context, fences []geofence.FenceItem, manifest *geofence.Manifest) error {
	// Store manifest
	if err := p.store.SetManifest(ctx, manifest); err != nil {
		return fmt.Errorf("failed to store manifest: %w", err)
	}

	// Update version
	if err := p.store.SetVersion(ctx, manifest.Version); err != nil {
		return fmt.Errorf("failed to set version: %w", err)
	}

	return nil
}

// writeManifest writes a manifest to a JSON file.
func (p *Publisher) writeManifest(manifest *geofence.Manifest, path string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// GetCurrentVersion returns the current version number.
func (p *Publisher) GetCurrentVersion(ctx context.Context) (uint64, error) {
	return p.store.GetVersion(ctx)
}

// ListFences returns all fences in the database.
func (p *Publisher) ListFences(ctx context.Context) ([]*geofence.FenceItem, error) {
	return p.store.ListFences(ctx)
}

// GetFence retrieves a fence by ID.
func (p *Publisher) GetFence(ctx context.Context, id string) (*geofence.FenceItem, error) {
	return p.store.GetFence(ctx, id)
}

// DeleteFence removes a fence from the database.
func (p *Publisher) DeleteFence(ctx context.Context, id string) error {
	return p.store.DeleteFence(ctx, id)
}

// Close closes the publisher and releases resources.
func (p *Publisher) Close() error {
	return p.store.Close()
}

// Initialize creates a new empty database.
func Initialize(ctx context.Context, cfg *config.PublisherConfig) error {
	// Determine store path
	storePath := "./geofence.db"
	if cfg.OutputDir != "" {
		storePath = filepath.Join(cfg.OutputDir, "geofence.db")
	}

	// Remove existing database if it exists
	if _, err := os.Stat(storePath); err == nil {
		os.Remove(storePath)
	}

	// Create new database
	store, err := storage.Open(ctx, &storage.Config{Path: storePath})
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer store.Close()

	// Initialize version to 0
	if err := store.SetVersion(ctx, 0); err != nil {
		return fmt.Errorf("failed to initialize version: %w", err)
	}

	return nil
}
