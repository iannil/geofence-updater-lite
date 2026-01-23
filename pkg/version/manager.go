// Package version provides version management for geofence updates,
// including Merkle tree generation, delta creation, and snapshot management.
package version

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/binarydiff"
	"github.com/iannil/geofence-updater-lite/pkg/crypto"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
	"github.com/iannil/geofence-updater-lite/pkg/merkle"
	"github.com/iannil/geofence-updater-lite/pkg/storage"
)

// Manager handles version management for geofence updates.
type Manager struct {
	store          *storage.SQLiteStore
	keyPair        *crypto.KeyPair
	currentVersion uint64
	mu             sync.RWMutex
	baseDir        string
}

// Config is the configuration for the version manager.
type Config struct {
	StorePath   string        // Path to the SQLite database
	PrivateKey  []byte        // Ed25519 private key for signing
	KeyID       string        // Key ID for the signature
	OutputDir   string        // Directory for output files
	CDNBaseURL  string        // Base URL for CDN uploads
}

// NewManager creates a new version manager.
func NewManager(ctx context.Context, cfg *Config) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Open storage
	store, err := storage.Open(ctx, &storage.Config{Path: cfg.StorePath})
	if err != nil {
		return nil, fmt.Errorf("failed to open storage: %w", err)
	}

	// Load key pair
	keyPair, err := crypto.DeriveKeyPair(nil, cfg.PrivateKey)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("invalid key pair: %w", err)
	}

	mgr := &Manager{
		store:     store,
		keyPair:   keyPair,
		baseDir:   cfg.OutputDir,
	}

	// Load current version
	version, err := store.GetVersion(ctx)
	if err != nil {
		// First time initialization
		version = 0
		if err := store.SetVersion(ctx, version); err != nil {
			store.Close()
			return nil, fmt.Errorf("failed to set initial version: %w", err)
		}
	}
	mgr.currentVersion = version

	return mgr, nil
}

// GetCurrentVersion returns the current version number.
func (m *Manager) GetCurrentVersion(ctx context.Context) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentVersion, nil
}

// PublishNewVersion creates a new version with the given fences.
func (m *Manager) PublishNewVersion(ctx context.Context, fences []geofence.FenceItem) (*PublishResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment version
	newVersion := m.currentVersion + 1

	// Build Merkle tree
	tree, err := merkle.NewTree(fences)
	if err != nil {
		return nil, fmt.Errorf("failed to build merkle tree: %w", err)
	}

	// Create snapshot
	snapshotData, snapshotSize, err := merkle.CreateSnapshot(fences)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Compute root hash
	rootHash := tree.RootHash()

	// Create manifest
	manifest := &geofence.Manifest{
		Version:      newVersion,
		Timestamp:   time.Now().Unix(),
		RootHash:     rootHash[:],
		SnapshotURL: fmt.Sprintf("/snapshots/v%d.bin", newVersion),
		SnapshotSize: uint64(snapshotSize),
		Message:     fmt.Sprintf("Version %d - %d fences", newVersion, len(fences)),
	}

	// Sign manifest
	manifestData, err := manifest.MarshalBinaryForSigning()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest for signing: %w", err)
	}
	signature, err := m.keyPair.Sign(manifestData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign manifest: %w", err)
	}
	manifest.SetSignature(signature, m.keyPair.KeyID)

	// Save manifest to storage
	if err := m.store.SetManifest(ctx, manifest); err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	// Update version
	if err := m.store.SetVersion(ctx, newVersion); err != nil {
		return nil, fmt.Errorf("failed to update version: %w", err)
	}
	m.currentVersion = newVersion

	// Write files to output directory
	snapshotPath := filepath.Join(m.baseDir, fmt.Sprintf("v%d.bin", newVersion))
	if err := os.WriteFile(snapshotPath, snapshotData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write snapshot: %w", err)
	}

	manifestPath := filepath.Join(m.baseDir, "manifest.json")
	if err := writeManifest(manifest, manifestPath); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	// If there's a previous version, compute delta
	var deltaPath string
	if newVersion > 1 {
		// Get old fences from storage
		oldFencePtrs, err := m.store.ListFences(ctx)
		if err == nil && len(oldFencePtrs) > 0 {
			// Convert pointers to values
			oldFences := make([]geofence.FenceItem, len(oldFencePtrs))
			for i, f := range oldFencePtrs {
				oldFences[i] = *f
			}

			// Compute delta
			delta, err := binarydiff.Diff(oldFences, fences)
			if err == nil {
				deltaData := delta.DiffData
				deltaPath = fmt.Sprintf("/patches/v%d_to_v%d.bin", newVersion-1, newVersion)

				// Write delta file
				deltaFullPath := filepath.Join(m.baseDir, deltaPath)
				if err := os.WriteFile(deltaFullPath, deltaData, 0644); err != nil {
					return nil, fmt.Errorf("failed to write delta: %w", err)
				}

				// Update manifest with delta info
				manifest.DeltaURL = deltaPath
				manifest.DeltaSize = uint64(delta.ToSize)

				// Use delta hash
				manifest.DeltaHash = delta.DiffHash

				// Update manifest file
				if err := writeManifest(manifest, manifestPath); err != nil {
					return nil, fmt.Errorf("failed to update manifest with delta info: %w", err)
				}
			}
		}
	}

	return &PublishResult{
		Version:     newVersion,
		Manifest:    manifest,
		SnapshotPath: snapshotPath,
		DeltaPath:   deltaPath,
	}, nil
}

// PublishResult contains the results of a publish operation.
type PublishResult struct {
	Version     uint64
	Manifest    *geofence.Manifest
	SnapshotPath string
	DeltaPath   string
}

// writeManifest writes a manifest to a JSON file.
func writeManifest(manifest *geofence.Manifest, path string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadVersion loads a specific version from storage.
func (m *Manager) LoadVersion(ctx context.Context, version uint64) ([]geofence.FenceItem, error) {
	// In a real system, this would load from a snapshot file
	// For now, we'll just return all fences from storage
	fencePtrs, err := m.store.ListFences(ctx)
	if err != nil {
		return nil, err
	}

	// Convert pointers to values
	fences := make([]geofence.FenceItem, len(fencePtrs))
	for i, f := range fencePtrs {
		fences[i] = *f
	}
	return fences, nil
}

// UpdateFence updates or adds a fence in the current version.
func (m *Manager) UpdateFence(ctx context.Context, fence *geofence.FenceItem) error {
	// Check if fence exists
	_, err := m.store.GetFence(ctx, fence.ID)
	if err != nil {
		// Fence doesn't exist, add it
		return m.store.AddFence(ctx, fence)
	}

	// Fence exists, update it
	return m.store.UpdateFence(ctx, fence)
}

// RemoveFence removes a fence from the current version.
func (m *Manager) RemoveFence(ctx context.Context, id string) error {
	return m.store.DeleteFence(ctx, id)
}

// QueryAtPoint checks if a location is allowed for flight.
func (m *Manager) QueryAtPoint(ctx context.Context, lat, lon float64) (bool, *geofence.FenceItem, error) {
	results, err := m.store.QueryAtPoint(ctx, lat, lon)
	if err != nil {
		return false, nil, fmt.Errorf("query failed: %w", err)
	}

	if len(results) == 0 {
		return true, nil, nil
	}

	// Return the highest priority restriction
	highestPriority := results[0]
	for i := 1; i < len(results); i++ {
		if results[i].Priority > highestPriority.Priority {
			highestPriority = results[i]
		}
	}

	switch highestPriority.Type {
	case geofence.FenceTypePermanentNoFly, geofence.FenceTypeTempRestriction:
		return false, highestPriority, nil
	case geofence.FenceTypeAltitudeLimit:
		return true, highestPriority, nil
	case geofence.FenceTypeSpeedLimit:
		return true, highestPriority, nil
	default:
		return true, nil, nil
	}
}

// Check is a convenience method for QueryAtPoint.
func (m *Manager) Check(ctx context.Context, lat, lon float64) (bool, *geofence.FenceItem, error) {
	return m.QueryAtPoint(ctx, lat, lon)
}

// GetFence retrieves a fence by ID.
func (m *Manager) GetFence(ctx context.Context, id string) (*geofence.FenceItem, error) {
	return m.store.GetFence(ctx, id)
}

// ListFences returns all fences.
func (m *Manager) ListFences(ctx context.Context) ([]*geofence.FenceItem, error) {
	return m.store.ListFences(ctx)
}

// GetManifest returns the current manifest.
func (m *Manager) GetManifest(ctx context.Context) (*geofence.Manifest, error) {
	return m.store.GetManifest(ctx)
}

// Sync updates the local database from the remote manifest.
func (m *Manager) Sync(ctx context.Context, remoteManifest *geofence.Manifest) (*SyncResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	remoteVer := remoteManifest.Version
	localVer := m.currentVersion // Direct access since we already hold the lock

	if remoteVer <= localVer {
		return &SyncResult{
			UpToDate: true,
		CurrentVersion: localVer,
			RemoteVersion: remoteVer,
		}, nil
	}

	// Need to update - compute delta
	// For now, we'll just update the manifest and re-download everything
	// In a complete implementation, this would download delta or snapshot

	return &SyncResult{
		UpToDate:      false,
		CurrentVersion: localVer,
		RemoteVersion: remoteVer,
		DeltaAvailable: false,
	}, nil
}

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	UpToDate      bool
	CurrentVersion uint64
	RemoteVersion uint64
	DeltaAvailable bool
}

// Close closes the version manager and releases resources.
func (m *Manager) Close() error {
	return m.store.Close()
}
