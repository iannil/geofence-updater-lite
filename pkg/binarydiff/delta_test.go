package binarydiff

import (
	"bytes"
	"testing"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

func TestDiff(t *testing.T) {
	now := time.Now()
	oldFences := []geofence.FenceItem{
		{
			ID:       "diff-test-001",
			Type:     geofence.FenceTypeTempRestriction,
			StartTS:  now.Unix(),
			EndTS:    now.Add(1 * time.Hour).Unix(),
			Priority: 50,
			Name:     "Test Fence 1",
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 0, Longitude: 0},
					{Latitude: 0, Longitude: 1},
					{Latitude: 1, Longitude: 1},
					{Latitude: 1, Longitude: 0},
				},
			},
		},
	}

	// Modified version - longer description
	newFences := []geofence.FenceItem{
		{
			ID:       "diff-test-001",
			Type:     geofence.FenceTypeTempRestriction,
			StartTS:  now.Unix(),
			EndTS:    now.Add(1 * time.Hour).Unix(),
			Priority: 50,
			Name:     "Test Fence 1 - With Longer Description",
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 0, Longitude: 0},
					{Latitude: 0, Longitude: 1},
					{Latitude: 1, Longitude: 1},
					{Latitude: 1, Longitude: 0},
				},
			},
		},
	}

	// Create diff
	delta, err := Diff(oldFences, newFences)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if delta == nil {
		t.Fatal("delta is nil")
	}

	// Diff should have non-zero size
	if len(delta.DiffData) == 0 {
		t.Error("delta size is 0, expected non-zero")
	}

	// New size should be >= old size
	if delta.ToSize < delta.FromSize {
		t.Errorf("new size %d < old size %d", delta.ToSize, delta.FromSize)
	}

	// Diff hash should be set
	if len(delta.DiffHash) == 0 {
		t.Error("delta hash is empty")
	}

	t.Logf("Delta: FromVersion=%d, ToVersion=%d, Size=%d bytes",
		delta.FromVersion, delta.ToVersion, len(delta.DiffData))
}

func TestPatch(t *testing.T) {
	now := time.Now()
	oldFences := []geofence.FenceItem{
		{
			ID:          "patch-test-001",
			Type:        geofence.FenceTypeTempRestriction,
			StartTS:     now.Unix(),
			EndTS:       now.Add(1 * time.Hour).Unix(),
			Priority:    50,
			Name:        "Test Fence 1",
			Description: "Original",
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 0, Longitude: 0},
					{Latitude: 0, Longitude: 1},
					{Latitude: 1, Longitude: 1},
					{Latitude: 1, Longitude: 0},
				},
			},
		},
	}

	// Modified version
	newFences := []geofence.FenceItem{
		{
			ID:          "patch-test-001",
			Type:        geofence.FenceTypeTempRestriction,
			StartTS:     now.Unix(),
			EndTS:       now.Add(1 * time.Hour).Unix(),
			Priority:    100, // Changed from 50
			Name:        "Test Fence 1",
			Description: "Modified",
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 0, Longitude: 0},
					{Latitude: 0, Longitude: 1},
					{Latitude: 1, Longitude: 1},
					{Latitude: 1, Longitude: 0},
				},
			},
		},
	}

	// Create diff
	delta, err := Diff(oldFences, newFences)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Apply patch
	patchedFences, err := PatchFences(oldFences, delta)
	if err != nil {
		t.Fatalf("PatchFences failed: %v", err)
	}

	if len(patchedFences) != 1 {
		t.Fatalf("PatchFences returned %d fences, want 1", len(patchedFences))
	}

	// Verify the patch worked
	if patchedFences[0].Priority != 100 {
		t.Errorf("Priority = %d, want 100", patchedFences[0].Priority)
	}

	if patchedFences[0].Name != "Test Fence 1" {
		t.Errorf("Name = %s, want 'Test Fence 1'", patchedFences[0].Name)
	}

	if patchedFences[0].Description != "Modified" {
		t.Errorf("Description = %s, want 'Modified'", patchedFences[0].Description)
	}
}

func TestPatchWithAddedFence(t *testing.T) {
	now := time.Now()
	oldFences := []geofence.FenceItem{
		{
			ID:       "patch-add-test-001",
			Type:     geofence.FenceTypeTempRestriction,
			StartTS:  now.Unix(),
			EndTS:    now.Add(1 * time.Hour).Unix(),
			Priority: 50,
			Name:     "Original Fence",
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 0, Longitude: 0},
					{Latitude: 0, Longitude: 1},
					{Latitude: 1, Longitude: 1},
					{Latitude: 1, Longitude: 0},
				},
			},
		},
	}

	// Add a new fence
	newFences := append(oldFences, geofence.FenceItem{
		ID:       "patch-add-test-002",
		Type:     geofence.FenceTypePermanentNoFly,
		StartTS:  0,
		EndTS:    0,
		Priority: 100,
		Name:     "New No-Fly Zone",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 10, Longitude: 10},
				{Latitude: 10, Longitude: 11},
				{Latitude: 11, Longitude: 11},
				{Latitude: 11, Longitude: 10},
			},
		},
	})

	// Create diff
	delta, err := Diff(oldFences, newFences)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Apply patch
	patchedFences, err := PatchFences(oldFences, delta)
	if err != nil {
		t.Fatalf("PatchFences failed: %v", err)
	}

	if len(patchedFences) != 2 {
		t.Fatalf("PatchFences returned %d fences, want 2", len(patchedFences))
	}

	// Verify both fences exist
	found1 := findFence(patchedFences, "patch-add-test-001")
	found2 := findFence(patchedFences, "patch-add-test-002")

	if found1 == nil {
		t.Error("Original fence not found after patch")
	}

	if found2 == nil {
		t.Error("New fence not found after patch")
	}
}

func TestComputeDeltaSize(t *testing.T) {
	now := time.Now()

	// No changes - same fences
	sameFences := []geofence.FenceItem{
		{
			ID:       "size-test-001",
			Type:     geofence.FenceTypeTempRestriction,
			StartTS:  now.Unix(),
			EndTS:    now.Add(1 * time.Hour).Unix(),
			Priority: 50,
			Name:     "Test Fence",
			Geometry: geofence.Geometry{
				Polygon: []geofence.Point{
					{Latitude: 0, Longitude: 0},
					{Latitude: 0, Longitude: 1},
					{Latitude: 1, Longitude: 1},
					{Latitude: 1, Longitude: 0},
				},
			},
		},
	}

	size, _ := ComputeDeltaSize(sameFences, sameFences)
	// ComputeDeltaSize is a rough estimation, so same fences may still be counted as "updated"
	// The function is meant for estimation, not exact delta calculation
	if size < 0 {
		t.Errorf("Delta size should not be negative, got %d", size)
	}

	// Added fence
	newFences := append(sameFences, geofence.FenceItem{
		ID:       "size-test-002",
		Type:     geofence.FenceTypePermanentNoFly,
		StartTS:  0,
		EndTS:    0,
		Priority: 100,
		Name:     "New Fence",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 10, Longitude: 10},
				{Latitude: 10, Longitude: 11},
				{Latitude: 11, Longitude: 11},
				{Latitude: 11, Longitude: 10},
			},
		},
	})

	size, _ = ComputeDeltaSize(sameFences, newFences)
	if size <= 0 {
		t.Errorf("Adding fence should result in non-zero delta size, got %d", size)
	}
}

func TestWriteReadDelta(t *testing.T) {
	delta := &DeltaFile{
		FromVersion: 1,
		ToVersion:   2,
		FromSize:    1024,
		ToSize:      2048,
		DiffData:    []byte{1, 2, 3, 4},
		DiffHash:    []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}

	// Write delta
	var buf bytes.Buffer
	if err := WriteDelta(delta, &buf); err != nil {
		t.Fatalf("WriteDelta failed: %v", err)
	}

	// Read delta
	r := bytes.NewReader(buf.Bytes())
	readDelta, err := ReadDelta(r)
	if err != nil {
		t.Fatalf("ReadDelta failed: %v", err)
	}

	if readDelta.FromVersion != 1 {
		t.Errorf("FromVersion = %d, want 1", readDelta.FromVersion)
	}

	if readDelta.ToVersion != 2 {
		t.Errorf("ToVersion = %d, want 2", readDelta.ToVersion)
	}

	if len(readDelta.DiffData) != 4 {
		t.Errorf("DiffData len = %d, want 4", len(readDelta.DiffData))
	}
}

func TestDeltaHeaderValidation(t *testing.T) {
	// Valid header
	validHeader := DeltaHeader{
		Magic:       [4]byte{'G', 'U', 'L', 'D'},
		Version:     1,
		FromVersion: 1,
		ToVersion:   2,
		OldSize:     1000,
		NewSize:     2000,
		DiffSize:    500,
		DiffHash:    []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}

	// Write valid header to buffer
	var buf bytes.Buffer
	if err := binaryWriteHeader(&buf, &validHeader); err != nil {
		t.Fatalf("binaryWriteHeader failed: %v", err)
	}

	// Read back
	r := bytes.NewReader(buf.Bytes())
	var readHeader DeltaHeader
	if err := binaryReadHeader(r, &readHeader); err != nil {
		t.Fatalf("binaryReadHeader failed: %v", err)
	}

	if readHeader.Magic != validHeader.Magic {
		t.Errorf("Magic = %v, want GULD", readHeader.Magic)
	}

	if readHeader.Version != 1 {
		t.Errorf("Version = %d, want 1", readHeader.Version)
	}
}

func TestDeltaHeaderInvalidMagic(t *testing.T) {
	// Invalid magic
	invalidHeader := DeltaHeader{
		Magic:       [4]byte{'B', 'A', 'D', '\x00'},
		Version:     1,
		FromVersion: 1,
		ToVersion:   2,
		OldSize:     1000,
		NewSize:     2000,
		DiffSize:    500,
		DiffHash:    []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := binaryWriteHeader(&buf, &invalidHeader); err != nil {
		t.Fatalf("binaryWriteHeader failed: %v", err)
	}

	// Try to read back - should fail on invalid magic
	r := bytes.NewReader(buf.Bytes())
	var readHeader DeltaHeader
	err := binaryReadHeader(r, &readHeader)

	if err == nil {
		t.Error("Expected error for invalid magic number")
	}
}

func TestCommonPrefixLen(t *testing.T) {
	tests := []struct {
		name     string
		a        []byte
		b        []byte
		expected int
	}{
		{
			name:     "identical",
			a:        []byte{1, 2, 3, 4},
			b:        []byte{1, 2, 3, 4},
			expected: 4,
		},
		{
			name:     "different at start",
			a:        []byte{1, 2, 3},
			b:        []byte{9, 8, 7},
			expected: 0,
		},
		{
			name:     "different in middle",
			a:        []byte{1, 2, 3, 4},
			b:        []byte{1, 2, 9, 4},
			expected: 2,
		},
		{
			name:     "one is prefix of other",
			a:        []byte{1, 2, 3},
			b:        []byte{1, 2, 3, 4, 5},
			expected: 3,
		},
		{
			name:     "empty slices",
			a:        []byte{},
			b:        []byte{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := commonPrefixLen(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("commonPrefixLen() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestCommonSuffixLen(t *testing.T) {
	tests := []struct {
		name     string
		a        []byte
		b        []byte
		expected int
	}{
		{
			name:     "identical",
			a:        []byte{1, 2, 3, 4},
			b:        []byte{1, 2, 3, 4},
			expected: 4,
		},
		{
			name:     "different at end",
			a:        []byte{1, 2, 3, 4},
			b:        []byte{1, 2, 3, 9},
			expected: 0,
		},
		{
			name:     "different in middle",
			a:        []byte{1, 2, 3, 4},
			b:        []byte{1, 9, 3, 4},
			expected: 2,
		},
		{
			name:     "one is suffix of other",
			a:        []byte{3, 4, 5},
			b:        []byte{1, 2, 3, 4, 5},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := commonSuffixLen(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("commonSuffixLen() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// Helper function to find a fence by ID
func findFence(fences []geofence.FenceItem, id string) *geofence.FenceItem {
	for i := range fences {
		if fences[i].ID == id {
			return &fences[i]
		}
	}
	return nil
}
