package merkle

import (
	"testing"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

func TestNewTree(t *testing.T) {
	now := time.Now()
	fences := []geofence.FenceItem{
		{
			ID:        "test-001",
			Type:      geofence.FenceTypeTempRestriction,
			StartTS:   now.Unix(),
			EndTS:     now.Add(1 * time.Hour).Unix(),
			Priority:  50,
			Name:      "Test Fence 1",
			Signature: []byte("sig1"),
		},
		{
			ID:        "test-002",
			Type:      geofence.FenceTypePermanentNoFly,
			StartTS:   0,
			EndTS:     0,
			Priority:  100,
			Name:      "Test Fence 2",
			Signature: []byte("sig2"),
		},
	}

	tree, err := NewTree(fences)
	if err != nil {
		t.Fatalf("NewTree failed: %v", err)
	}

	if tree == nil {
		t.Fatal("tree is nil")
	}

	// Check root hash is computed
	rootHash := tree.RootHash()
	if len(rootHash) == 0 {
		t.Error("root hash is empty")
	}
}

func TestEmptyTree(t *testing.T) {
	tree, err := NewTree([]geofence.FenceItem{})
	if err != nil {
		t.Fatalf("NewTree with empty fences failed: %v", err)
	}

	if tree == nil {
		t.Fatal("tree is nil")
	}

	rootHash := tree.RootHash()
	// Empty tree should have all-zero hash
	emptyHash := Hash{}
	if rootHash != emptyHash {
		t.Errorf("expected empty root hash, got %x", rootHash)
	}
}

func TestRootHash_Deterministic(t *testing.T) {
	now := time.Now()
	fences := []geofence.FenceItem{
		{
			ID:        "det-001",
			Type:      geofence.FenceTypeTempRestriction,
			StartTS:   now.Unix(),
			EndTS:     now.Add(1 * time.Hour).Unix(),
			Priority:  50,
			Name:      "Deterministic Test",
			Signature: []byte("sig"),
		},
	}

	tree1, err := NewTree(fences)
	if err != nil {
		t.Fatalf("NewTree failed: %v", err)
	}

	tree2, err := NewTree(fences)
	if err != nil {
		t.Fatalf("NewTree failed: %v", err)
	}

	hash1 := tree1.RootHash()
	hash2 := tree2.RootHash()

	if hash1 != hash2 {
		t.Errorf("root hashes differ: %x != %x", hash1, hash2)
	}
}

func TestGetProof(t *testing.T) {
	now := time.Now()
	fences := []geofence.FenceItem{
		{
			ID:        "proof-001",
			Type:      geofence.FenceTypeTempRestriction,
			StartTS:   now.Unix(),
			EndTS:     now.Add(1 * time.Hour).Unix(),
			Priority:  50,
			Name:      "Proof Test",
			Signature: []byte("sig"),
		},
	}

	tree, err := NewTree(fences)
	if err != nil {
		t.Fatalf("NewTree failed: %v", err)
	}

	proof, err := tree.GetProof("proof-001")
	if err != nil {
		t.Fatalf("GetProof failed: %v", err)
	}

	// For a single node tree, proof should be empty
	if len(proof) != 0 {
		t.Errorf("expected empty proof for single node, got %d elements", len(proof))
	}
}

func TestGetProof_NotFound(t *testing.T) {
	tree, err := NewTree([]geofence.FenceItem{})
	if err != nil {
		t.Fatalf("NewTree failed: %v", err)
	}

	_, err = tree.GetProof("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent fence")
	}
}

func TestComputeDelta(t *testing.T) {
	now := time.Now()
	oldFences := []geofence.FenceItem{
		{
			ID:       "delta-001",
			Type:     geofence.FenceTypeTempRestriction,
			StartTS:  now.Unix(),
			EndTS:    now.Add(1 * time.Hour).Unix(),
			Priority: 50,
			Name:     "Old Fence",
		},
	}

	newFences := []geofence.FenceItem{
		{
			ID:       "delta-001",
			Type:     geofence.FenceTypeTempRestriction,
			StartTS:  now.Unix(),
			EndTS:    now.Add(2 * time.Hour).Unix(),
			Priority: 100,
			Name:     "Updated Fence",
		},
		{
			ID:       "delta-002",
			Type:     geofence.FenceTypePermanentNoFly,
			StartTS:  0,
			EndTS:    0,
			Priority: 100,
			Name:     "New Fence",
		},
	}

	delta, size, err := ComputeDelta(oldFences, newFences)
	if err != nil {
		t.Fatalf("ComputeDelta failed: %v", err)
	}

	if len(delta) == 0 {
		t.Error("delta is empty")
	}

	if size == 0 {
		t.Error("size is 0")
	}
}

func TestCreateSnapshot(t *testing.T) {
	now := time.Now()
	fences := []geofence.FenceItem{
		{
			ID:       "snap-001",
			Type:     geofence.FenceTypeTempRestriction,
			StartTS:  now.Unix(),
			EndTS:    now.Add(1 * time.Hour).Unix(),
			Priority: 50,
			Name:     "Snapshot Test",
		},
	}

	data, size, err := CreateSnapshot(fences)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("snapshot data is empty")
	}

	if size == 0 {
		t.Error("size is 0")
	}

	// Load and verify
	loadedFences, err := LoadSnapshot(data)
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	if len(loadedFences) != len(fences) {
		t.Errorf("loaded %d fences, expected %d", len(loadedFences), len(fences))
	}
}
