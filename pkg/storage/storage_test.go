package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "test.db")
}

func TestOpen(t *testing.T) {
	ctx := context.Background()
	path := tempDB(t)

	store, err := Open(ctx, &Config{Path: path})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Check tables exist
	var tableName string
	err = store.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='fences'").Scan(&tableName)
	if err != nil {
		t.Errorf("fences table not found: %v", err)
	}

	err = store.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='fence_index'").Scan(&tableName)
	if err != nil {
		t.Errorf("fence_index table not found: %v", err)
	}
}

func TestAddAndGetFence(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	now := time.Now()
	fence := &geofence.FenceItem{
		ID:     "test-001",
		Type:   geofence.FenceTypeTempRestriction,
		StartTS: now.Unix(),
		EndTS:   now.Add(24 * time.Hour).Unix(),
		Priority: 50,
		Name:     "Test Fence",
		Description: "Test fence for storage",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 39.0, Longitude: 116.0},
				{Latitude: 39.0, Longitude: 117.0},
				{Latitude: 40.0, Longitude: 117.0},
				{Latitude: 40.0, Longitude: 116.0},
			},
		},
	}

	// Add fence
	err = store.AddFence(ctx, fence)
	if err != nil {
		t.Fatalf("AddFence failed: %v", err)
	}

	// Get fence
	retrieved, err := store.GetFence(ctx, "test-001")
	if err != nil {
		t.Fatalf("GetFence failed: %v", err)
	}

	if retrieved.ID != fence.ID {
		t.Errorf("ID = %s, want %s", retrieved.ID, fence.ID)
	}
	if retrieved.Type != fence.Type {
		t.Errorf("Type = %d, want %d", retrieved.Type, fence.Type)
	}
	if retrieved.Name != fence.Name {
		t.Errorf("Name = %s, want %s", retrieved.Name, fence.Name)
	}
	if len(retrieved.Geometry.Polygon) != len(fence.Geometry.Polygon) {
		t.Errorf("Polygon length = %d, want %d", len(retrieved.Geometry.Polygon), len(fence.Geometry.Polygon))
	}
}

func TestUpdateFence(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	fence := &geofence.FenceItem{
		ID:     "test-002",
		Type:   geofence.FenceTypePermanentNoFly,
		StartTS: 0,
		EndTS:   0,
		Priority: 50,
		Name:     "Original Name",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 39.0, Longitude: 116.0},
				{Latitude: 40.0, Longitude: 116.0},
				{Latitude: 40.0, Longitude: 117.0},
				{Latitude: 39.0, Longitude: 117.0},
			},
		},
	}

	err = store.AddFence(ctx, fence)
	if err != nil {
		t.Fatalf("AddFence failed: %v", err)
	}

	// Update fence
	fence.Priority = 100
	fence.Name = "Updated Name"
	err = store.UpdateFence(ctx, fence)
	if err != nil {
		t.Fatalf("UpdateFence failed: %v", err)
	}

	// Verify update
	retrieved, err := store.GetFence(ctx, "test-002")
	if err != nil {
		t.Fatalf("GetFence failed: %v", err)
	}

	if retrieved.Priority != 100 {
		t.Errorf("Priority = %d, want 100", retrieved.Priority)
	}
	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %s, want 'Updated Name'", retrieved.Name)
	}
}

func TestDeleteFence(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	fence := &geofence.FenceItem{
		ID:     "test-003",
		Type:   geofence.FenceTypeTempRestriction,
		StartTS: time.Now().Unix(),
		EndTS:   time.Now().Add(1 * time.Hour).Unix(),
		Priority: 50,
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 0, Longitude: 0},
				{Latitude: 1, Longitude: 0},
				{Latitude: 1, Longitude: 1},
				{Latitude: 0, Longitude: 1},
			},
		},
	}

	err = store.AddFence(ctx, fence)
	if err != nil {
		t.Fatalf("AddFence failed: %v", err)
	}

	err = store.DeleteFence(ctx, "test-003")
	if err != nil {
		t.Fatalf("DeleteFence failed: %v", err)
	}

	// Verify deleted
	_, err = store.GetFence(ctx, "test-003")
	if err == nil {
		t.Error("expected error when getting deleted fence")
	}
}

func TestListFences(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	now := time.Now()
	// Add multiple fences with unique IDs
	fences := []*geofence.FenceItem{
		{
			ID:     "fence-list-1",
			Type:   geofence.FenceTypeTempRestriction,
			Priority: 10,
			StartTS: now.Unix(),
			EndTS:   now.Add(1 * time.Hour).Unix(),
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 0, Longitude: 0},
					{Latitude: 1, Longitude: 0},
					{Latitude: 1, Longitude: 1},
					{Latitude: 0, Longitude: 1},
				},
			},
		},
		{
			ID:     "fence-list-2",
			Type:   geofence.FenceTypePermanentNoFly,
			Priority: 100,
			StartTS: 0,
			EndTS:   0,
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 10, Longitude: 10},
					{Latitude: 11, Longitude: 10},
					{Latitude: 11, Longitude: 11},
					{Latitude: 10, Longitude: 11},
				},
			},
		},
	}

	for _, f := range fences {
		if err := store.AddFence(ctx, f); err != nil {
			t.Fatalf("AddFence failed: %v", err)
		}
	}

	// List fences
	list, err := store.ListFences(ctx)
	if err != nil {
		t.Fatalf("ListFences failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("ListFences returned %d fences, want 2", len(list))
	}

	// Check priority ordering (higher priority first)
	if list[0].Priority < list[1].Priority {
		t.Error("fences not ordered by priority DESC")
	}
}

func TestQueryAtPoint(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Add a fence covering Beijing area with active time window
	now := time.Now()
	fence := &geofence.FenceItem{
		ID:     "beijing-query-test",
		Type:   geofence.FenceTypeTempRestriction,
		StartTS: now.Add(-1 * time.Hour).Unix(), // Started 1 hour ago
		EndTS:   now.Add(24 * time.Hour).Unix(), // Ends in 24 hours
		Priority: 50,
		Name:     "Beijing Restriction",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 39.9, Longitude: 116.4},
				{Latitude: 39.9, Longitude: 116.5},
				{Latitude: 40.0, Longitude: 116.5},
				{Latitude: 40.0, Longitude: 116.4},
			},
		},
	}

	err = store.AddFence(ctx, fence)
	if err != nil {
		t.Fatalf("AddFence failed: %v", err)
	}

	// Query point inside fence
	results, err := store.QueryAtPoint(ctx, 39.95, 116.45)
	if err != nil {
		t.Fatalf("QueryAtPoint failed: %v", err)
	}

	if len(results) == 0 {
		t.Errorf("QueryAtPoint returned %d results, want at least 1", len(results))
	} else {
		if results[0].ID != "beijing-query-test" {
			t.Errorf("QueryAtPoint returned fence %s, want beijing-query-test", results[0].ID)
		}
	}

	// Query point outside fence
	results, err = store.QueryAtPoint(ctx, 0, 0)
	if err != nil {
		t.Fatalf("QueryAtPoint failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("QueryAtPoint outside fence returned %d results, want 0", len(results))
	}
}

func TestQueryInBounds(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Add fences in different locations
	fences := []*geofence.FenceItem{
		{
			ID:     "fence-north",
			Type:   geofence.FenceTypeTempRestriction,
			StartTS: time.Now().Unix(),
			EndTS:   time.Now().Add(1 * time.Hour).Unix(),
			Priority: 50,
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 40.0, Longitude: 116.0},
					{Latitude: 40.0, Longitude: 117.0},
					{Latitude: 41.0, Longitude: 117.0},
					{Latitude: 41.0, Longitude: 116.0},
				},
			},
		},
		{
			ID:     "fence-south",
			Type:   geofence.FenceTypeTempRestriction,
			StartTS: time.Now().Unix(),
			EndTS:   time.Now().Add(1 * time.Hour).Unix(),
			Priority: 50,
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 38.0, Longitude: 116.0},
					{Latitude: 38.0, Longitude: 117.0},
					{Latitude: 39.0, Longitude: 117.0},
					{Latitude: 39.0, Longitude: 116.0},
				},
			},
		},
	}

	for _, f := range fences {
		if err := store.AddFence(ctx, f); err != nil {
			t.Fatalf("AddFence failed: %v", err)
		}
	}

	// Query bounds covering north area
	bounds := &geofence.BoundingBox{
		MinLat: 40.0,
		MaxLat: 41.0,
		MinLon: 116.0,
		MaxLon: 117.0,
	}

	results, err := store.QueryInBounds(ctx, bounds)
	if err != nil {
		t.Fatalf("QueryInBounds failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("QueryInBounds returned no results")
	}

	// Check we got the north fence
	found := false
	for _, r := range results {
		if r.ID == "fence-north" {
			found = true
			break
		}
	}
	if !found {
		t.Error("fence-north not found in QueryInBounds results")
	}
}

func TestManifestStorage(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	manifest := &geofence.Manifest{
		Version:     1,
		Timestamp:   time.Now().Unix(),
		DeltaURL:    "/patches/v1.bin",
		SnapshotURL: "/snapshots/v1.bin",
		Message:     "Test manifest",
	}

	// Set manifest
	err = store.SetManifest(ctx, manifest)
	if err != nil {
		t.Fatalf("SetManifest failed: %v", err)
	}

	// Get manifest
	retrieved, err := store.GetManifest(ctx)
	if err != nil {
		t.Fatalf("GetManifest failed: %v", err)
	}

	if retrieved.Version != manifest.Version {
		t.Errorf("Version = %d, want %d", retrieved.Version, manifest.Version)
	}
	if retrieved.Message != manifest.Message {
		t.Errorf("Message = %s, want %s", retrieved.Message, manifest.Message)
	}
}

func TestVersionStorage(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Set version
	err = store.SetVersion(ctx, 42)
	if err != nil {
		t.Fatalf("SetVersion failed: %v", err)
	}

	// Get version
	version, err := store.GetVersion(ctx)
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}

	if version != 42 {
		t.Errorf("Version = %d, want 42", version)
	}
}

func TestTransaction(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Begin transaction
	tx, err := store.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	// Add fence within transaction
	fence := &geofence.FenceItem{
		ID:     "tx-fence",
		Type:   geofence.FenceTypeTempRestriction,
		StartTS: time.Now().Unix(),
		EndTS:   time.Now().Add(1 * time.Hour).Unix(),
		Priority: 50,
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 0, Longitude: 0},
				{Latitude: 1, Longitude: 0},
				{Latitude: 1, Longitude: 1},
				{Latitude: 0, Longitude: 1},
			},
		},
	}

	err = store.AddFence(ctx, fence)
	if err != nil {
		t.Fatalf("AddFence failed: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify fence was added
	_, err = store.GetFence(ctx, "tx-fence")
	if err != nil {
		t.Errorf("GetFence after commit failed: %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, &Config{Path: tempDB(t)})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			_, err := store.GetVersion(ctx)
			if err != nil {
				t.Errorf("concurrent read %d failed: %v", n, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent test timeout")
		}
	}
}
