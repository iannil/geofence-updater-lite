package geofence

import (
	"testing"
	"time"
)

// Test fixtures

func sampleManifest() *Manifest {
	return &Manifest{
		Version:     1,
		Timestamp:   time.Now().Unix(),
		DeltaURL:    "/patches/v0_to_v1.bin",
		SnapshotURL: "/snapshots/v1.bin",
		DeltaSize:   1024,
		SnapshotSize: 4096,
		Message:     "Initial release",
	}
}

func sampleFences() []FenceItem {
	now := time.Now()
	return []FenceItem{
		{
			ID:     "test-temp-001",
			Type:   FenceTypeTempRestriction,
			StartTS: now.Unix(),
			EndTS:   now.Add(24 * time.Hour).Unix(),
			Priority: 50,
			Name:     "Test Temporary Restriction",
			Description: "Temporary no-fly zone for testing",
			Geometry: Geometry{
				Polygon: []Point{
					{Latitude: 39.9042, Longitude: 116.4074},
					{Latitude: 39.9142, Longitude: 116.4074},
					{Latitude: 39.9142, Longitude: 116.4174},
					{Latitude: 39.9042, Longitude: 116.4174},
				},
			},
		},
	}
}

func temporaryFence() FenceItem {
	now := time.Now()
	return FenceItem{
		ID:     "test-temp-001",
		Type:   FenceTypeTempRestriction,
		StartTS: now.Unix(),
		EndTS:   now.Add(24 * time.Hour).Unix(),
		Priority: 50,
		Name:     "Test Temporary Restriction",
		Description: "Temporary no-fly zone for testing",
		Geometry: Geometry{
			Polygon: []Point{
				{Latitude: 39.9042, Longitude: 116.4074},
				{Latitude: 39.9142, Longitude: 116.4074},
				{Latitude: 39.9142, Longitude: 116.4174},
				{Latitude: 39.9042, Longitude: 116.4174},
			},
		},
	}
}

func permanentNoFlyZone() FenceItem {
	return FenceItem{
		ID:     "test-perm-001",
		Type:   FenceTypePermanentNoFly,
		StartTS: 0,
		EndTS:   0,
		Priority: 100,
		Name:     "Test Airport No-Fly Zone",
		Description: "Permanent restriction around airport",
		Geometry: Geometry{
			Polygon: []Point{
				{Latitude: 31.1443, Longitude: 121.8083},
				{Latitude: 31.1543, Longitude: 121.8083},
				{Latitude: 31.1543, Longitude: 121.8183},
				{Latitude: 31.1443, Longitude: 121.8183},
			},
		},
	}
}

func altitudeLimitFence() FenceItem {
	return FenceItem{
		ID:     "test-alt-001",
		Type:   FenceTypeAltitudeLimit,
		StartTS: 0,
		EndTS:   0,
		Priority: 30,
		MaxAltitude: 120,
		Name:     "Test Altitude Limit",
		Description: "Maximum altitude 120m",
		Geometry: Geometry{
			Polygon: []Point{
				{Latitude: 22.5431, Longitude: 114.0579},
				{Latitude: 22.5531, Longitude: 114.0579},
				{Latitude: 22.5531, Longitude: 114.0679},
				{Latitude: 22.5431, Longitude: 114.0679},
			},
		},
	}
}

func TestManifest_MarshalBinaryForSigning(t *testing.T) {
	manifest := sampleManifest()

	data, err := manifest.MarshalBinaryForSigning()
	if err != nil {
		t.Fatalf("MarshalBinaryForSigning failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("marshaled data should not be empty")
	}

	// The signature field should not be included in the signing data
	if len(data) < 50 {
		t.Error("marshaled data seems too short")
	}
}

func TestManifest_SetSignature(t *testing.T) {
	manifest := sampleManifest()

	sig := []byte{1, 2, 3, 4, 5}
	keyID := "test-key-id"

	manifest.SetSignature(sig, keyID)

	if string(manifest.Signature) != string(sig) {
		t.Error("signature not set correctly")
	}

	if manifest.KeyID != keyID {
		t.Error("key ID not set correctly")
	}
}

func TestComputeRootHash(t *testing.T) {
	t.Run("empty fences", func(t *testing.T) {
		hash, err := ComputeRootHash([]FenceItem{})
		if err != nil {
			t.Fatalf("ComputeRootHash failed: %v", err)
		}
		if len(hash) != 0 {
			t.Errorf("empty hash = %v, want empty", hash)
		}
	})

	t.Run("single fence", func(t *testing.T) {
		fences := []FenceItem{temporaryFence()}
		hash, err := ComputeRootHash(fences)
		if err != nil {
			t.Fatalf("ComputeRootHash failed: %v", err)
		}
		if len(hash) != 32 { // SHA-256 output size
			t.Errorf("hash length = %d, want 32", len(hash))
		}
	})

	t.Run("multiple fences", func(t *testing.T) {
		fences := sampleFences()
		hash1, err := ComputeRootHash(fences)
		if err != nil {
			t.Fatalf("ComputeRootHash failed: %v", err)
		}

		// Same fences should produce same hash
		hash2, err := ComputeRootHash(fences)
		if err != nil {
			t.Fatalf("ComputeRootHash failed: %v", err)
		}

		if string(hash1) != string(hash2) {
			t.Error("same fences should produce same hash")
		}
	})
}

func TestApplyDelta(t *testing.T) {
	t.Run("add fence", func(t *testing.T) {
		existing := []FenceItem{}
		delta := FenceDelta{
			Added: []FenceItem{temporaryFence()},
		}

		result, err := ApplyDelta(existing, delta)
		if err != nil {
			t.Fatalf("ApplyDelta failed: %v", err)
		}

		if len(result) != 1 {
			t.Errorf("result length = %d, want 1", len(result))
		}
	})

	t.Run("remove fence", func(t *testing.T) {
		existing := []FenceItem{temporaryFence()}
		delta := FenceDelta{
			RemovedIDs: []string{temporaryFence().ID},
		}

		result, err := ApplyDelta(existing, delta)
		if err != nil {
			t.Fatalf("ApplyDelta failed: %v", err)
		}

		if len(result) != 0 {
			t.Errorf("result length = %d, want 0", len(result))
		}
	})

	t.Run("update fence", func(t *testing.T) {
		fence := temporaryFence()
		existing := []FenceItem{fence}

		updated := fence
		updated.Priority = 999

		delta := FenceDelta{
			Updated: []FenceItem{updated},
		}

		result, err := ApplyDelta(existing, delta)
		if err != nil {
			t.Fatalf("ApplyDelta failed: %v", err)
		}

		if len(result) != 1 {
			t.Errorf("result length = %d, want 1", len(result))
		}

		if result[0].Priority != 999 {
			t.Errorf("priority = %d, want 999", result[0].Priority)
		}
	})

	t.Run("add duplicate fence", func(t *testing.T) {
		fence := temporaryFence()
		existing := []FenceItem{fence}
		delta := FenceDelta{
			Added: []FenceItem{fence},
		}

		_, err := ApplyDelta(existing, delta)
		if err == nil {
			t.Error("expected error when adding duplicate fence")
		}
	})

	t.Run("update non-existent fence", func(t *testing.T) {
		existing := []FenceItem{}
		delta := FenceDelta{
			Updated: []FenceItem{temporaryFence()},
		}

		_, err := ApplyDelta(existing, delta)
		if err == nil {
			t.Error("expected error when updating non-existent fence")
		}
	})
}

func TestCreateDelta(t *testing.T) {
	oldFences := []FenceItem{
		temporaryFence(),
		permanentNoFlyZone(),
	}

	// Create a modified version
	newFences := make([]FenceItem, len(oldFences))
	copy(newFences, oldFences)

	// Modify one fence
	newFences[0].Priority = 999

	// Remove one
	newFences = append(newFences[:1], newFences[2:]...)

	// Add one
	newFences = append(newFences, altitudeLimitFence())

	delta := CreateDelta(oldFences, newFences)

	if len(delta.Updated) != 1 {
		t.Errorf("updated count = %d, want 1", len(delta.Updated))
	}

	if len(delta.RemovedIDs) != 1 {
		t.Errorf("removed count = %d, want 1", len(delta.RemovedIDs))
	}

	if len(delta.Added) != 1 {
		t.Errorf("added count = %d, want 1", len(delta.Added))
	}
}

func TestUpdaterConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &UpdaterConfig{
			ManifestURL: "https://example.com/manifest.json",
			PublicKey:   make([]byte, 32),
			StorePath:   "/data/geofence.db",
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate failed: %v", err)
		}
	})

	t.Run("missing manifest URL", func(t *testing.T) {
		cfg := &UpdaterConfig{
			PublicKey: make([]byte, 32),
			StorePath: "/data/geofence.db",
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for missing manifest URL")
		}
	})

	t.Run("missing public key", func(t *testing.T) {
		cfg := &UpdaterConfig{
			ManifestURL: "https://example.com/manifest.json",
			StorePath:   "/data/geofence.db",
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for missing public key")
		}
	})

	t.Run("missing store path", func(t *testing.T) {
		cfg := &UpdaterConfig{
			ManifestURL: "https://example.com/manifest.json",
			PublicKey:   make([]byte, 32),
		}
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for missing store path")
		}
	})

	t.Run("default values applied", func(t *testing.T) {
		cfg := &UpdaterConfig{
			ManifestURL: "https://example.com/manifest.json",
			PublicKey:   make([]byte, 32),
			StorePath:   "/data/geofence.db",
		}

		cfg.Validate()

		if cfg.SyncInterval == 0 {
			t.Error("sync interval should have default value")
		}

		if cfg.HTTPTimeout == 0 {
			t.Error("http timeout should have default value")
		}

		if cfg.MaxDownloadSize == 0 {
			t.Error("max download size should have default value")
		}
	})
}
