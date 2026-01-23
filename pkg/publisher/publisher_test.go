package publisher

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/config"
	"github.com/iannil/geofence-updater-lite/pkg/crypto"
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

func testConfig(t *testing.T) *config.PublisherConfig {
	t.Helper()

	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	return &config.PublisherConfig{
		PrivateKeyHex: crypto.MarshalPrivateKeyHex(kp.PrivateKey),
		KeyID:         kp.KeyID,
		OutputDir:     t.TempDir(),
		CDNBaseURL:    "https://cdn.example.com/geofence",
	}
}

func TestNewPublisher(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	if pub.keyPair == nil {
		t.Error("keyPair should not be nil")
	}
	if pub.store == nil {
		t.Error("store should not be nil")
	}
}

func TestNewPublisher_InvalidPrivateKey(t *testing.T) {
	ctx := context.Background()
	cfg := &config.PublisherConfig{
		PrivateKeyHex: "invalid-hex!",
		OutputDir:     t.TempDir(),
	}

	_, err := NewPublisher(ctx, cfg)
	if err == nil {
		t.Error("expected error for invalid private key")
	}
}

func TestPublish_SingleVersion(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	fences := []geofence.FenceItem{
		{
			ID:       "fence-001",
			Type:     geofence.FenceTypePermanentNoFly,
			StartTS:  0,
			EndTS:    0,
			Priority: 100,
			Name:     "Test Fence 1",
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

	result, err := pub.Publish(ctx, fences)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if result.Version != 1 {
		t.Errorf("Version = %d, want 1", result.Version)
	}
	if result.FencesCount != 1 {
		t.Errorf("FencesCount = %d, want 1", result.FencesCount)
	}

	// Verify manifest file was created
	manifestPath := filepath.Join(cfg.OutputDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("manifest.json was not created")
	}

	// Verify snapshot file was created
	snapshotPath := filepath.Join(cfg.OutputDir, "v1.bin")
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		t.Error("v1.bin was not created")
	}
}

func TestPublish_MultipleVersions(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	// First version
	fences1 := []geofence.FenceItem{
		{
			ID:       "fence-001",
			Type:     geofence.FenceTypePermanentNoFly,
			Priority: 100,
			Name:     "First Fence",
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

	result1, err := pub.Publish(ctx, fences1)
	if err != nil {
		t.Fatalf("First Publish failed: %v", err)
	}
	if result1.Version != 1 {
		t.Errorf("First Version = %d, want 1", result1.Version)
	}

	// Second version with additional fence
	fences2 := []geofence.FenceItem{
		fences1[0],
		{
			ID:       "fence-002",
			Type:     geofence.FenceTypeTempRestriction,
			StartTS:  time.Now().Unix(),
			EndTS:    time.Now().Add(24 * time.Hour).Unix(),
			Priority: 50,
			Name:     "Second Fence",
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 38.0, Longitude: 115.0},
					{Latitude: 38.0, Longitude: 116.0},
					{Latitude: 39.0, Longitude: 116.0},
					{Latitude: 39.0, Longitude: 115.0},
				},
			},
		},
	}

	result2, err := pub.Publish(ctx, fences2)
	if err != nil {
		t.Fatalf("Second Publish failed: %v", err)
	}
	if result2.Version != 2 {
		t.Errorf("Second Version = %d, want 2", result2.Version)
	}
	if result2.FencesCount != 2 {
		t.Errorf("FencesCount = %d, want 2", result2.FencesCount)
	}
}

func TestPublish_VerifyManifestContent(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	fences := []geofence.FenceItem{
		{
			ID:       "fence-verify",
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

	_, err = pub.Publish(ctx, fences)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Read and verify manifest
	manifestPath := filepath.Join(cfg.OutputDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var manifest geofence.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	if manifest.Version != 1 {
		t.Errorf("manifest.Version = %d, want 1", manifest.Version)
	}
	if manifest.SnapshotURL == "" {
		t.Error("manifest.SnapshotURL should not be empty")
	}
	if len(manifest.Signature) == 0 {
		t.Error("manifest should be signed")
	}
	if manifest.KeyID == "" {
		t.Error("manifest.KeyID should not be empty")
	}
}

func TestSignAndAdd(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	fence := &geofence.FenceItem{
		ID:       "sign-add-fence",
		Type:     geofence.FenceTypeTempRestriction,
		StartTS:  time.Now().Unix(),
		EndTS:    time.Now().Add(1 * time.Hour).Unix(),
		Priority: 50,
		Name:     "Signed Fence",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 0, Longitude: 0},
				{Latitude: 1, Longitude: 0},
				{Latitude: 1, Longitude: 1},
				{Latitude: 0, Longitude: 1},
			},
		},
	}

	err = pub.SignAndAdd(ctx, fence)
	if err != nil {
		t.Fatalf("SignAndAdd failed: %v", err)
	}

	// Verify fence was signed
	if len(fence.Signature) == 0 {
		t.Error("fence should be signed")
	}
	if fence.KeyID == "" {
		t.Error("fence.KeyID should not be empty")
	}

	// Verify fence was added to storage
	retrieved, err := pub.GetFence(ctx, "sign-add-fence")
	if err != nil {
		t.Fatalf("GetFence failed: %v", err)
	}
	if retrieved.Name != fence.Name {
		t.Errorf("Name = %s, want %s", retrieved.Name, fence.Name)
	}
}

func TestSignAndUpdate(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	fence := &geofence.FenceItem{
		ID:       "update-fence",
		Type:     geofence.FenceTypeTempRestriction,
		StartTS:  time.Now().Unix(),
		EndTS:    time.Now().Add(1 * time.Hour).Unix(),
		Priority: 50,
		Name:     "Original Name",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 0, Longitude: 0},
				{Latitude: 1, Longitude: 0},
				{Latitude: 1, Longitude: 1},
				{Latitude: 0, Longitude: 1},
			},
		},
	}

	// First add the fence
	err = pub.SignAndAdd(ctx, fence)
	if err != nil {
		t.Fatalf("SignAndAdd failed: %v", err)
	}

	// Update the fence
	fence.Name = "Updated Name"
	fence.Priority = 100
	err = pub.SignAndUpdate(ctx, fence)
	if err != nil {
		t.Fatalf("SignAndUpdate failed: %v", err)
	}

	// Verify update
	retrieved, err := pub.GetFence(ctx, "update-fence")
	if err != nil {
		t.Fatalf("GetFence failed: %v", err)
	}
	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %s, want 'Updated Name'", retrieved.Name)
	}
	if retrieved.Priority != 100 {
		t.Errorf("Priority = %d, want 100", retrieved.Priority)
	}
}

func TestDeleteFence(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	fence := &geofence.FenceItem{
		ID:       "delete-fence",
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

	err = pub.SignAndAdd(ctx, fence)
	if err != nil {
		t.Fatalf("SignAndAdd failed: %v", err)
	}

	err = pub.DeleteFence(ctx, "delete-fence")
	if err != nil {
		t.Fatalf("DeleteFence failed: %v", err)
	}

	_, err = pub.GetFence(ctx, "delete-fence")
	if err == nil {
		t.Error("expected error when getting deleted fence")
	}
}

func TestListFences(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	fences := []*geofence.FenceItem{
		{
			ID:       "list-fence-1",
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
			ID:       "list-fence-2",
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
		if err := pub.SignAndAdd(ctx, f); err != nil {
			t.Fatalf("SignAndAdd failed: %v", err)
		}
	}

	list, err := pub.ListFences(ctx)
	if err != nil {
		t.Fatalf("ListFences failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("ListFences returned %d fences, want 2", len(list))
	}
}

func TestGetCurrentVersion(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	// Initial version should be 0
	version, err := pub.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 0 {
		t.Errorf("initial version = %d, want 0", version)
	}

	// Publish and check version
	fences := []geofence.FenceItem{
		{
			ID:       "version-fence",
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

	_, err = pub.Publish(ctx, fences)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	version, err = pub.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 1 {
		t.Errorf("version after publish = %d, want 1", version)
	}
}

func TestInitialize(t *testing.T) {
	ctx := context.Background()

	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	cfg := &config.PublisherConfig{
		PrivateKeyHex: crypto.MarshalPrivateKeyHex(kp.PrivateKey),
		OutputDir:     t.TempDir(),
		CDNBaseURL:    "https://cdn.example.com/geofence",
	}

	err = Initialize(ctx, cfg)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify database was created
	dbPath := filepath.Join(cfg.OutputDir, "geofence.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database was not created")
	}

	// Create publisher and verify version is 0
	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}
	defer pub.Close()

	version, err := pub.GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion failed: %v", err)
	}
	if version != 0 {
		t.Errorf("version = %d, want 0", version)
	}
}

func TestPublisher_Close(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)

	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPublisher failed: %v", err)
	}

	err = pub.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
