package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/config"
	"github.com/iannil/geofence-updater-lite/pkg/crypto"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
	"github.com/iannil/geofence-updater-lite/pkg/merkle"
	"github.com/iannil/geofence-updater-lite/pkg/storage"
)

func testSyncerConfig(t *testing.T, serverURL string) *config.ClientConfig {
	t.Helper()

	return &config.ClientConfig{
		ManifestURL:        serverURL + "/manifest.json",
		HTTPTimeout:        5 * time.Second,
		UserAgent:          "test-syncer/1.0",
		StorePath:          filepath.Join(t.TempDir(), "sync.db"),
		InsecureSkipVerify: true,
	}
}

func TestNewSyncer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := &geofence.Manifest{Version: 1}
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	if syncer.client == nil {
		t.Error("client should not be nil")
	}
	if syncer.store == nil {
		t.Error("store should not be nil")
	}
}

func TestNewSyncer_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	cfg := &config.ClientConfig{
		// Missing required fields
	}

	_, err := NewSyncer(ctx, cfg)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestCheckForUpdates(t *testing.T) {
	expectedManifest := &geofence.Manifest{
		Version:     5,
		Timestamp:   time.Now().Unix(),
		SnapshotURL: "/snapshot.bin",
		Message:     "Test update",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(expectedManifest)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	manifest, err := syncer.CheckForUpdates(ctx)
	if err != nil {
		t.Fatalf("CheckForUpdates failed: %v", err)
	}

	if manifest.Version != expectedManifest.Version {
		t.Errorf("Version = %d, want %d", manifest.Version, expectedManifest.Version)
	}
	if manifest.Message != expectedManifest.Message {
		t.Errorf("Message = %s, want %s", manifest.Message, expectedManifest.Message)
	}

	// Verify lastCheck was updated
	lastCheck := syncer.GetLastCheckTime()
	if lastCheck.IsZero() {
		t.Error("lastCheck should not be zero")
	}
}

func TestSync_UpToDate(t *testing.T) {
	manifest := &geofence.Manifest{
		Version:   0,
		Timestamp: time.Now().Unix(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	result := syncer.Sync(ctx)

	if result.Error != nil {
		t.Fatalf("Sync failed: %v", result.Error)
	}
	if !result.UpToDate {
		t.Error("expected up to date")
	}
}

func TestSync_Snapshot(t *testing.T) {
	// Create test fences
	fences := []geofence.FenceItem{
		{
			ID:       "sync-fence-1",
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

	// Create snapshot
	snapshotData, _, err := merkle.CreateSnapshot(fences)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Create manifest
	tree, err := merkle.NewTree(fences)
	if err != nil {
		t.Fatalf("NewTree failed: %v", err)
	}
	rootHash := tree.RootHash()

	manifest := &geofence.Manifest{
		Version:      1,
		Timestamp:    time.Now().Unix(),
		SnapshotURL:  "/snapshot.bin",
		RootHash:     rootHash[:],
		SnapshotHash: crypto.ComputeSHA256(snapshotData),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			json.NewEncoder(w).Encode(manifest)
		} else if r.URL.Path == "/snapshot.bin" {
			w.Write(snapshotData)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	result := syncer.Sync(ctx)

	if result.Error != nil {
		t.Fatalf("Sync failed: %v", result.Error)
	}
	if result.UpToDate {
		t.Error("expected not up to date initially")
	}
	if result.CurrentVer != 1 {
		t.Errorf("CurrentVer = %d, want 1", result.CurrentVer)
	}
}

func TestSync_ManifestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	result := syncer.Sync(ctx)

	if result.Error == nil {
		t.Error("expected error for server error")
	}
}

func TestGetCurrentVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&geofence.Manifest{Version: 1})
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	version := syncer.GetCurrentVersion()
	if version != 0 {
		t.Errorf("initial version = %d, want 0", version)
	}
}

func TestGetLastCheckTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&geofence.Manifest{Version: 1})
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	// Initially zero
	lastCheck := syncer.GetLastCheckTime()
	if !lastCheck.IsZero() {
		t.Error("initial lastCheck should be zero")
	}

	// After check
	_, err = syncer.CheckForUpdates(ctx)
	if err != nil {
		t.Fatalf("CheckForUpdates failed: %v", err)
	}

	lastCheck = syncer.GetLastCheckTime()
	if lastCheck.IsZero() {
		t.Error("lastCheck should not be zero after check")
	}
}

func TestGetLastSyncTime(t *testing.T) {
	manifest := &geofence.Manifest{
		Version:   0,
		Timestamp: time.Now().Unix(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	// Initially zero
	lastSync := syncer.GetLastSyncTime()
	if !lastSync.IsZero() {
		t.Error("initial lastSync should be zero")
	}
}

func TestGetFences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&geofence.Manifest{Version: 0})
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	// Add fence directly to storage for testing
	store, err := storage.Open(ctx, &storage.Config{Path: cfg.StorePath})
	if err != nil {
		t.Fatalf("Open storage failed: %v", err)
	}
	fence := &geofence.FenceItem{
		ID:       "get-fences-test",
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
	if err := store.AddFence(ctx, fence); err != nil {
		t.Fatalf("AddFence failed: %v", err)
	}
	store.Close()

	// Reconnect syncer
	syncer, err = NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	fences, err := syncer.GetFences(ctx)
	if err != nil {
		t.Fatalf("GetFences failed: %v", err)
	}
	if len(fences) != 1 {
		t.Errorf("GetFences returned %d fences, want 1", len(fences))
	}
}

func TestCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&geofence.Manifest{Version: 0})
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	// Add a no-fly zone
	store, err := storage.Open(ctx, &storage.Config{Path: cfg.StorePath})
	if err != nil {
		t.Fatalf("Open storage failed: %v", err)
	}

	now := time.Now()
	fence := &geofence.FenceItem{
		ID:       "check-test",
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
	if err := store.AddFence(ctx, fence); err != nil {
		t.Fatalf("AddFence failed: %v", err)
	}
	store.Close()

	// Reconnect syncer
	syncer, err = NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	// Check inside no-fly zone
	allowed, restriction, err := syncer.Check(ctx, 39.5, 116.5)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if allowed {
		t.Error("expected not allowed inside no-fly zone")
	}
	if restriction == nil {
		t.Error("expected restriction to be returned")
	}

	// Check outside fence
	allowed, restriction, err = syncer.Check(ctx, 0, 0)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !allowed {
		t.Error("expected allowed outside fence")
	}
	if restriction != nil {
		t.Error("expected no restriction outside fence")
	}
}

func TestCheck_TempRestriction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&geofence.Manifest{Version: 0})
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	// Add a temp restriction
	store, err := storage.Open(ctx, &storage.Config{Path: cfg.StorePath})
	if err != nil {
		t.Fatalf("Open storage failed: %v", err)
	}

	now := time.Now()
	fence := &geofence.FenceItem{
		ID:       "temp-check",
		Type:     geofence.FenceTypeTempRestriction,
		StartTS:  now.Add(-1 * time.Hour).Unix(),
		EndTS:    now.Add(24 * time.Hour).Unix(),
		Priority: 50,
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 30.0, Longitude: 110.0},
				{Latitude: 30.0, Longitude: 111.0},
				{Latitude: 31.0, Longitude: 111.0},
				{Latitude: 31.0, Longitude: 110.0},
			},
		},
	}
	if err := store.AddFence(ctx, fence); err != nil {
		t.Fatalf("AddFence failed: %v", err)
	}
	store.Close()

	// Reconnect syncer
	syncer, err = NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	// Check inside temp restriction
	allowed, restriction, err := syncer.Check(ctx, 30.5, 110.5)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if allowed {
		t.Error("expected not allowed inside temp restriction")
	}
	if restriction == nil {
		t.Error("expected restriction to be returned")
	}
	if restriction.Type != geofence.FenceTypeTempRestriction {
		t.Errorf("Type = %d, want %d", restriction.Type, geofence.FenceTypeTempRestriction)
	}
}

func TestStartAutoSync(t *testing.T) {
	manifest := &geofence.Manifest{
		Version:   0,
		Timestamp: time.Now().Unix(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}
	defer syncer.Close()

	results := syncer.StartAutoSync(ctx, 100*time.Millisecond)

	// Get first result
	select {
	case result := <-results:
		if result.Error != nil {
			t.Errorf("initial sync error: %v", result.Error)
		}
		if !result.UpToDate {
			t.Error("expected up to date initially")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for sync result")
	}

	// Cancel and verify channel closes
	cancel()
	select {
	case _, ok := <-results:
		if ok {
			// Drain remaining results
			for range results {
			}
		}
	case <-time.After(2 * time.Second):
		t.Log("channel may still have pending results")
	}
}

func TestSyncer_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&geofence.Manifest{Version: 0})
	}))
	defer server.Close()

	ctx := context.Background()
	cfg := testSyncerConfig(t, server.URL)

	syncer, err := NewSyncer(ctx, cfg)
	if err != nil {
		t.Fatalf("NewSyncer failed: %v", err)
	}

	err = syncer.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestSyncResult_Fields(t *testing.T) {
	result := &SyncResult{
		UpToDate:      true,
		PreviousVer:   1,
		CurrentVer:    2,
		FencesAdded:   5,
		FencesRemoved: 2,
		FencesUpdated: 3,
		BytesDownload: 1024,
		Duration:      100 * time.Millisecond,
		Error:         nil,
	}

	if !result.UpToDate {
		t.Error("UpToDate should be true")
	}
	if result.PreviousVer != 1 {
		t.Errorf("PreviousVer = %d, want 1", result.PreviousVer)
	}
	if result.CurrentVer != 2 {
		t.Errorf("CurrentVer = %d, want 2", result.CurrentVer)
	}
}
