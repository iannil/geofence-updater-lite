package version

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/crypto"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

func testManagerConfig(t *testing.T) *Config {
	t.Helper()

	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	dir := t.TempDir()
	return &Config{
		StorePath:  filepath.Join(dir, "test.db"),
		PrivateKey: kp.PrivateKey,
		KeyID:      kp.KeyID,
		OutputDir:  dir,
	}
}

func TestNewManager(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	if mgr.store == nil {
		t.Error("store should not be nil")
	}
	if mgr.keyPair == nil {
		t.Error("keyPair should not be nil")
	}
}

func TestNewManager_NilConfig(t *testing.T) {
	ctx := context.Background()
	_, err := NewManager(ctx, nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestNewManager_InvalidPrivateKey(t *testing.T) {
	ctx := context.Background()
	cfg := &Config{
		StorePath:  filepath.Join(t.TempDir(), "test.db"),
		PrivateKey: []byte("invalid"),
		OutputDir:  t.TempDir(),
	}

	_, err := NewManager(ctx, cfg)
	if err == nil {
		t.Error("expected error for invalid private key")
	}
}

func TestGetCurrentVersion(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	version, err := mgr.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 0 {
		t.Errorf("initial version = %d, want 0", version)
	}
}

func TestPublishNewVersion(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	fences := []geofence.FenceItem{
		{
			ID:       "fence-001",
			Type:     geofence.FenceTypePermanentNoFly,
			Priority: 100,
			Name:     "Test Fence",
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 39.0, Longitude: 116.0},
					{Latitude: 39.0, Longitude: 117.0},
					{Latitude: 40.0, Longitude: 117.0},
					{Latitude: 40.0, Longitude: 116.0},
				},
			},
		},
	}

	result, err := mgr.PublishNewVersion(ctx, fences)
	if err != nil {
		t.Fatalf("PublishNewVersion failed: %v", err)
	}

	if result.Version != 1 {
		t.Errorf("Version = %d, want 1", result.Version)
	}
	if result.Manifest == nil {
		t.Error("Manifest should not be nil")
	}
	if result.SnapshotPath == "" {
		t.Error("SnapshotPath should not be empty")
	}

	// Verify snapshot file exists
	if _, err := os.Stat(result.SnapshotPath); os.IsNotExist(err) {
		t.Error("snapshot file was not created")
	}

	// Verify version was updated
	version, err := mgr.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 1 {
		t.Errorf("version after publish = %d, want 1", version)
	}
}

func TestPublishNewVersion_Multiple(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	// First version
	fences1 := []geofence.FenceItem{
		{
			ID:       "fence-001",
			Type:     geofence.FenceTypePermanentNoFly,
			Priority: 100,
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 39.0, Longitude: 116.0},
					{Latitude: 40.0, Longitude: 117.0},
					{Latitude: 40.0, Longitude: 116.0},
				},
			},
		},
	}

	result1, err := mgr.PublishNewVersion(ctx, fences1)
	if err != nil {
		t.Fatalf("First PublishNewVersion failed: %v", err)
	}
	if result1.Version != 1 {
		t.Errorf("First version = %d, want 1", result1.Version)
	}

	// Second version
	fences2 := append(fences1, geofence.FenceItem{
		ID:       "fence-002",
		Type:     geofence.FenceTypeTempRestriction,
		StartTS:  time.Now().Unix(),
		EndTS:    time.Now().Add(1 * time.Hour).Unix(),
		Priority: 50,
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 38.0, Longitude: 115.0},
				{Latitude: 39.0, Longitude: 116.0},
				{Latitude: 39.0, Longitude: 115.0},
			},
		},
	})

	result2, err := mgr.PublishNewVersion(ctx, fences2)
	if err != nil {
		t.Fatalf("Second PublishNewVersion failed: %v", err)
	}
	if result2.Version != 2 {
		t.Errorf("Second version = %d, want 2", result2.Version)
	}
}

func TestUpdateFence(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	fence := &geofence.FenceItem{
		ID:       "update-test",
		Type:     geofence.FenceTypePermanentNoFly,
		Priority: 50,
		Name:     "Original",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 0, Longitude: 0},
				{Latitude: 1, Longitude: 0},
				{Latitude: 1, Longitude: 1},
				{Latitude: 0, Longitude: 1},
			},
		},
	}

	// Add fence
	err = mgr.UpdateFence(ctx, fence)
	if err != nil {
		t.Fatalf("UpdateFence (add) failed: %v", err)
	}

	// Verify added
	retrieved, err := mgr.GetFence(ctx, "update-test")
	if err != nil {
		t.Fatalf("GetFence failed: %v", err)
	}
	if retrieved.Name != "Original" {
		t.Errorf("Name = %s, want 'Original'", retrieved.Name)
	}

	// Update fence
	fence.Name = "Updated"
	fence.Priority = 100
	err = mgr.UpdateFence(ctx, fence)
	if err != nil {
		t.Fatalf("UpdateFence (update) failed: %v", err)
	}

	// Verify updated
	retrieved, err = mgr.GetFence(ctx, "update-test")
	if err != nil {
		t.Fatalf("GetFence failed: %v", err)
	}
	if retrieved.Name != "Updated" {
		t.Errorf("Name = %s, want 'Updated'", retrieved.Name)
	}
	if retrieved.Priority != 100 {
		t.Errorf("Priority = %d, want 100", retrieved.Priority)
	}
}

func TestRemoveFence(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	fence := &geofence.FenceItem{
		ID:       "remove-test",
		Type:     geofence.FenceTypePermanentNoFly,
		Priority: 100,
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 0, Longitude: 0},
				{Latitude: 1, Longitude: 1},
				{Latitude: 0, Longitude: 1},
			},
		},
	}

	err = mgr.UpdateFence(ctx, fence)
	if err != nil {
		t.Fatalf("UpdateFence failed: %v", err)
	}

	err = mgr.RemoveFence(ctx, "remove-test")
	if err != nil {
		t.Fatalf("RemoveFence failed: %v", err)
	}

	_, err = mgr.GetFence(ctx, "remove-test")
	if err == nil {
		t.Error("expected error when getting removed fence")
	}
}

func TestQueryAtPoint(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	now := time.Now()
	fence := &geofence.FenceItem{
		ID:       "query-test",
		Type:     geofence.FenceTypePermanentNoFly,
		StartTS:  now.Add(-1 * time.Hour).Unix(),
		EndTS:    now.Add(24 * time.Hour).Unix(),
		Priority: 100,
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 39.0, Longitude: 116.0},
				{Latitude: 39.0, Longitude: 117.0},
				{Latitude: 40.0, Longitude: 117.0},
				{Latitude: 40.0, Longitude: 116.0},
			},
		},
	}

	err = mgr.UpdateFence(ctx, fence)
	if err != nil {
		t.Fatalf("UpdateFence failed: %v", err)
	}

	// Query inside fence
	allowed, restriction, err := mgr.QueryAtPoint(ctx, 39.5, 116.5)
	if err != nil {
		t.Fatalf("QueryAtPoint failed: %v", err)
	}
	if allowed {
		t.Error("expected not allowed inside no-fly zone")
	}
	if restriction == nil {
		t.Error("expected restriction to be returned")
	}

	// Query outside fence
	allowed, restriction, err = mgr.QueryAtPoint(ctx, 0, 0)
	if err != nil {
		t.Fatalf("QueryAtPoint failed: %v", err)
	}
	if !allowed {
		t.Error("expected allowed outside fence")
	}
	if restriction != nil {
		t.Error("expected no restriction outside fence")
	}
}

func TestCheck(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	now := time.Now()
	fence := &geofence.FenceItem{
		ID:       "check-test",
		Type:     geofence.FenceTypeTempRestriction,
		StartTS:  now.Add(-1 * time.Hour).Unix(),
		EndTS:    now.Add(24 * time.Hour).Unix(),
		Priority: 50,
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 38.0, Longitude: 115.0},
				{Latitude: 38.0, Longitude: 116.0},
				{Latitude: 39.0, Longitude: 116.0},
				{Latitude: 39.0, Longitude: 115.0},
			},
		},
	}

	err = mgr.UpdateFence(ctx, fence)
	if err != nil {
		t.Fatalf("UpdateFence failed: %v", err)
	}

	// Check should be equivalent to QueryAtPoint
	allowed, restriction, err := mgr.Check(ctx, 38.5, 115.5)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if allowed {
		t.Error("expected not allowed inside temp restriction")
	}
	if restriction == nil {
		t.Error("expected restriction to be returned")
	}
}

func TestQueryAtPoint_AltitudeLimit(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	now := time.Now()
	fence := &geofence.FenceItem{
		ID:          "altitude-test",
		Type:        geofence.FenceTypeAltitudeLimit,
		StartTS:     now.Add(-1 * time.Hour).Unix(),
		EndTS:       now.Add(24 * time.Hour).Unix(),
		Priority:    50,
		MaxAltitude: 120,
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 30.0, Longitude: 110.0},
				{Latitude: 30.0, Longitude: 111.0},
				{Latitude: 31.0, Longitude: 111.0},
				{Latitude: 31.0, Longitude: 110.0},
			},
		},
	}

	err = mgr.UpdateFence(ctx, fence)
	if err != nil {
		t.Fatalf("UpdateFence failed: %v", err)
	}

	// Altitude limit should allow flight (with restriction)
	allowed, restriction, err := mgr.QueryAtPoint(ctx, 30.5, 110.5)
	if err != nil {
		t.Fatalf("QueryAtPoint failed: %v", err)
	}
	if !allowed {
		t.Error("expected allowed with altitude limit")
	}
	if restriction == nil {
		t.Error("expected altitude restriction to be returned")
	}
	if restriction.MaxAltitude != 120 {
		t.Errorf("MaxAltitude = %d, want 120", restriction.MaxAltitude)
	}
}

func TestListFences(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	fences := []*geofence.FenceItem{
		{
			ID:       "list-1",
			Type:     geofence.FenceTypePermanentNoFly,
			Priority: 100,
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
			ID:       "list-2",
			Type:     geofence.FenceTypeTempRestriction,
			StartTS:  time.Now().Unix(),
			EndTS:    time.Now().Add(1 * time.Hour).Unix(),
			Priority: 50,
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
		if err := mgr.UpdateFence(ctx, f); err != nil {
			t.Fatalf("UpdateFence failed: %v", err)
		}
	}

	list, err := mgr.ListFences(ctx)
	if err != nil {
		t.Fatalf("ListFences failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListFences returned %d fences, want 2", len(list))
	}
}

func TestGetManifest(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	// Initially no manifest
	_, err = mgr.GetManifest(ctx)
	if err == nil {
		t.Log("No manifest initially, which is expected")
	}

	// Publish a version
	fences := []geofence.FenceItem{
		{
			ID:       "manifest-test",
			Type:     geofence.FenceTypePermanentNoFly,
			Priority: 100,
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 0, Longitude: 0},
					{Latitude: 1, Longitude: 1},
					{Latitude: 0, Longitude: 1},
				},
			},
		},
	}

	_, err = mgr.PublishNewVersion(ctx, fences)
	if err != nil {
		t.Fatalf("PublishNewVersion failed: %v", err)
	}

	// Now manifest should exist
	manifest, err := mgr.GetManifest(ctx)
	if err != nil {
		t.Fatalf("GetManifest failed: %v", err)
	}
	if manifest.Version != 1 {
		t.Errorf("manifest.Version = %d, want 1", manifest.Version)
	}
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	// Sync with newer version
	remoteManifest := &geofence.Manifest{
		Version:   5,
		Timestamp: time.Now().Unix(),
	}

	result, err := mgr.Sync(ctx, remoteManifest)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if result.UpToDate {
		t.Error("expected not up to date")
	}
	if result.CurrentVersion != 0 {
		t.Errorf("CurrentVersion = %d, want 0", result.CurrentVersion)
	}
	if result.RemoteVersion != 5 {
		t.Errorf("RemoteVersion = %d, want 5", result.RemoteVersion)
	}
}

func TestSync_AlreadyUpToDate(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	// Publish a version
	fences := []geofence.FenceItem{
		{
			ID:       "sync-test",
			Type:     geofence.FenceTypePermanentNoFly,
			Priority: 100,
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 0, Longitude: 0},
					{Latitude: 1, Longitude: 1},
					{Latitude: 0, Longitude: 1},
				},
			},
		},
	}

	_, err = mgr.PublishNewVersion(ctx, fences)
	if err != nil {
		t.Fatalf("PublishNewVersion failed: %v", err)
	}

	// Sync with same version
	remoteManifest := &geofence.Manifest{
		Version:   1,
		Timestamp: time.Now().Unix(),
	}

	result, err := mgr.Sync(ctx, remoteManifest)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if !result.UpToDate {
		t.Error("expected up to date")
	}
}

func TestLoadVersion(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	fence := &geofence.FenceItem{
		ID:       "load-test",
		Type:     geofence.FenceTypePermanentNoFly,
		Priority: 100,
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 0, Longitude: 0},
				{Latitude: 1, Longitude: 0},
				{Latitude: 1, Longitude: 1},
				{Latitude: 0, Longitude: 1},
			},
		},
	}

	err = mgr.UpdateFence(ctx, fence)
	if err != nil {
		t.Fatalf("UpdateFence failed: %v", err)
	}

	fences, err := mgr.LoadVersion(ctx, 1)
	if err != nil {
		t.Fatalf("LoadVersion failed: %v", err)
	}

	if len(fences) != 1 {
		t.Errorf("LoadVersion returned %d fences, want 1", len(fences))
	}
	if fences[0].ID != "load-test" {
		t.Errorf("fence ID = %s, want 'load-test'", fences[0].ID)
	}
}

func TestManager_Close(t *testing.T) {
	ctx := context.Background()
	cfg := testManagerConfig(t)

	mgr, err := NewManager(ctx, cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
