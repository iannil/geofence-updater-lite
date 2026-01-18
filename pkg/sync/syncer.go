// Package sync provides geofence synchronization functionality
// for keeping local databases up-to-date with remote sources.
package sync

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/binarydiff"
	"github.com/iannil/geofence-updater-lite/pkg/client"
	"github.com/iannil/geofence-updater-lite/pkg/config"
	"github.com/iannil/geofence-updater-lite/pkg/crypto"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
	"github.com/iannil/geofence-updater-lite/pkg/merkle"
	"github.com/iannil/geofence-updater-lite/pkg/storage"
)

// Syncer handles synchronization of geofence data from a remote source.
type Syncer struct {
	client       *client.Client
	store        *storage.SQLiteStore
	cfg          *config.ClientConfig
	currentVer   uint64
	lastCheck    time.Time
	lastSyncTime time.Time
}

// NewSyncer creates a new geofence syncer.
func NewSyncer(ctx context.Context, cfg *config.ClientConfig) (*Syncer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create HTTP client
	httpClient, err := client.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Open storage
	store, err := storage.Open(ctx, &storage.Config{Path: cfg.StorePath})
	if err != nil {
		return nil, fmt.Errorf("failed to open storage: %w", err)
	}

	// Load current version
	currentVer, err := store.GetVersion(ctx)
	if err != nil {
		currentVer = 0 // First time
	}

	return &Syncer{
		client:     httpClient,
		store:      store,
		cfg:        cfg,
		currentVer: currentVer,
		lastCheck:  time.Time{},
	}, nil
}

// SyncResult contains the result of a sync operation.
type SyncResult struct {
	UpToDate      bool
	PreviousVer   uint64
	CurrentVer    uint64
	FencesAdded   int
	FencesRemoved int
	FencesUpdated int
	BytesDownload int
	Duration      time.Duration
	Error         error
}

// CheckForUpdates checks if there's a new version available without downloading.
func (s *Syncer) CheckForUpdates(ctx context.Context) (*geofence.Manifest, error) {
	s.lastCheck = time.Now()

	manifest, err := s.client.FetchManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	return manifest, nil
}

// Sync performs a full synchronization with the remote source.
func (s *Syncer) Sync(ctx context.Context) *SyncResult {
	start := time.Now()
	result := &SyncResult{
		PreviousVer: s.currentVer,
	}

	// Fetch remote manifest
	manifest, err := s.client.FetchManifest(ctx)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch manifest: %w", err)
		return result
	}

	result.CurrentVer = manifest.Version

	// Check if update is needed
	if manifest.Version <= s.currentVer {
		result.UpToDate = true
		return result
	}

	// Need to update
	log.Printf("[Sync] New version available: %d -> %d", s.currentVer, manifest.Version)

	// Decide whether to use delta or snapshot
	useDelta := (manifest.Version - s.currentVer) == 1 && manifest.DeltaURL != ""

	if useDelta {
		log.Printf("[Sync] Using delta update from %s", manifest.DeltaURL)
		err = s.applyDelta(ctx, manifest)
	} else {
		log.Printf("[Sync] Using snapshot from %s", manifest.SnapshotURL)
		err = s.applySnapshot(ctx, manifest)
	}

	if err != nil {
		result.Error = fmt.Errorf("failed to apply update: %w", err)
		return result
	}

	// Update current version
	s.currentVer = manifest.Version
	s.lastSyncTime = time.Now()
	result.Duration = time.Since(start)

	log.Printf("[Sync] Sync complete: version %d in %v", manifest.Version, result.Duration)

	return result
}

// applyDelta applies a delta update to the local fence database.
func (s *Syncer) applyDelta(ctx context.Context, manifest *geofence.Manifest) error {
	// Fetch delta data
	deltaData, err := s.client.FetchDelta(ctx, manifest.DeltaURL)
	if err != nil {
		return fmt.Errorf("failed to fetch delta: %w", err)
	}

	// Verify delta hash
	if len(manifest.DeltaHash) > 0 {
		if !crypto.VerifyHash(deltaData, manifest.DeltaHash) {
			return fmt.Errorf("delta hash verification failed")
		}
	}

	// Get current fences
	oldFences, err := s.getCurrentFences(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current fences: %w", err)
	}

	// Parse delta file
	delta, err := binarydiff.ReadDelta(bytes.NewReader(deltaData))
	if err != nil {
		return fmt.Errorf("failed to parse delta: %w", err)
	}

	// Fill version info
	delta.FromVersion = s.currentVer
	delta.ToVersion = manifest.Version

	// Apply patch
	newFences, err := binarydiff.PatchFences(oldFences, delta)
	if err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}

	// Update storage
	if err := s.updateStorage(ctx, newFences, manifest); err != nil {
		return fmt.Errorf("failed to update storage: %w", err)
	}

	return nil
}

// applySnapshot applies a full snapshot update to the local fence database.
func (s *Syncer) applySnapshot(ctx context.Context, manifest *geofence.Manifest) error {
	// Fetch snapshot data
	snapshotData, err := s.client.FetchSnapshot(ctx, manifest.SnapshotURL)
	if err != nil {
		return fmt.Errorf("failed to fetch snapshot: %w", err)
	}

	// Verify snapshot hash
	if len(manifest.SnapshotHash) > 0 {
		if !crypto.VerifyHash(snapshotData, manifest.SnapshotHash) {
			return fmt.Errorf("snapshot hash verification failed")
		}
	}

	// Load snapshot
	fences, err := merkle.LoadSnapshot(snapshotData)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Verify Merkle root hash
	if len(manifest.RootHash) > 0 {
		tree, err := merkle.NewTree(fences)
		if err != nil {
			return fmt.Errorf("failed to build Merkle tree: %w", err)
		}
		rootHash := tree.RootHash()
		if !crypto.VerifyHash(rootHash[:], manifest.RootHash) {
			return fmt.Errorf("root hash verification failed")
		}
	}

	// Update storage
	if err := s.updateStorage(ctx, fences, manifest); err != nil {
		return fmt.Errorf("failed to update storage: %w", err)
	}

	return nil
}

// getCurrentFences retrieves all current fences from storage.
func (s *Syncer) getCurrentFences(ctx context.Context) ([]geofence.FenceItem, error) {
	fencePtrs, err := s.store.ListFences(ctx)
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
func (s *Syncer) updateStorage(ctx context.Context, fences []geofence.FenceItem, manifest *geofence.Manifest) error {
	// Begin transaction
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing fences
	// Note: In a production system, this would be more sophisticated
	// For now, we'll delete all and re-add
	for _, f := range fences {
		// Try to update first
		err := s.store.UpdateFence(ctx, &f)
		if err != nil {
			// Doesn't exist, add it
			if err := s.store.AddFence(ctx, &f); err != nil {
				return fmt.Errorf("failed to add fence %s: %w", f.ID, err)
			}
		}
	}

	// Store manifest
	if err := s.store.SetManifest(ctx, manifest); err != nil {
		return fmt.Errorf("failed to store manifest: %w", err)
	}

	// Update version
	if err := s.store.SetVersion(ctx, manifest.Version); err != nil {
		return fmt.Errorf("failed to set version: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// StartAutoSync starts automatic synchronization in the background.
func (s *Syncer) StartAutoSync(ctx context.Context, interval time.Duration) <-chan *SyncResult {
	results := make(chan *SyncResult, 1)

	go func() {
		defer close(results)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Do initial sync
		result := s.Sync(ctx)
		select {
		case results <- result:
		case <-ctx.Done():
			return
		}

		for {
			select {
			case <-ticker.C:
				result := s.Sync(ctx)
				select {
				case results <- result:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return results
}

// GetFences retrieves all current fences.
func (s *Syncer) GetFences(ctx context.Context) ([]geofence.FenceItem, error) {
	return s.getCurrentFences(ctx)
}

// Check checks if a location is allowed for flight.
func (s *Syncer) Check(ctx context.Context, lat, lon float64) (bool, *geofence.FenceItem, error) {
	results, err := s.store.QueryAtPoint(ctx, lat, lon)
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
	default:
		return true, highestPriority, nil
	}
}

// GetCurrentVersion returns the current version number.
func (s *Syncer) GetCurrentVersion() uint64 {
	return s.currentVer
}

// GetLastCheckTime returns the time of the last update check.
func (s *Syncer) GetLastCheckTime() time.Time {
	return s.lastCheck
}

// GetLastSyncTime returns the time of the last successful sync.
func (s *Syncer) GetLastSyncTime() time.Time {
	return s.lastSyncTime
}

// Close closes the syncer and releases resources.
func (s *Syncer) Close() error {
	return s.store.Close()
}
