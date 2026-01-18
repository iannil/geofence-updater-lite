package geofence

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// MarshalBinary serializes the manifest to bytes for signing.
// This excludes the Signature field itself.
func (m *Manifest) MarshalBinaryForSigning() ([]byte, error) {
	copy := *m
	copy.Signature = nil

	data, err := json.Marshal(copy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	return data, nil
}

// SetSignature sets the signature on the manifest.
func (m *Manifest) SetSignature(sig []byte, keyID string) {
	m.Signature = sig
	m.KeyID = keyID
}

// VerifySignature verifies the manifest's signature.
// This is a placeholder - actual verification is done by the crypto package.
func (m *Manifest) VerifySignature(publicKey []byte) bool {
	// Actual implementation in crypto package
	return len(m.Signature) > 0
}

// ComputeRootHash computes a hash from a collection of fences.
// This is a simplified version - full Merkle tree implementation comes later.
func ComputeRootHash(fences []FenceItem) ([]byte, error) {
	if len(fences) == 0 {
		return []byte{}, nil
	}

	h := sha256.New()
	for _, f := range fences {
		fenceHash, err := hashFenceItem(f)
		if err != nil {
			return nil, err
		}
		h.Write(fenceHash)
	}
	return h.Sum(nil), nil
}

// hashFenceItem creates a hash of a fence item for the Merkle tree.
func hashFenceItem(f FenceItem) ([]byte, error) {
	h := sha256.New()

	// Hash all fields except signature
	fmt.Fprintf(h, "%s|%d|%d|%d", f.ID, f.Type, f.StartTS, f.EndTS)
	fmt.Fprintf(h, "|%d|%d|%d|%s|%s",
		f.Priority, f.MaxAltitude, f.MaxSpeed, f.Name, f.Description)

	// Hash geometry
	if len(f.Geometry.Polygon) > 0 {
		for _, p := range f.Geometry.Polygon {
			fmt.Fprintf(h, "|p%f,%f", p.Latitude, p.Longitude)
		}
	}
	if f.Geometry.CircleCenter != nil {
		fmt.Fprintf(h, "|c%f,%f,%f", f.Geometry.CircleCenter.Latitude,
			f.Geometry.CircleCenter.Longitude, f.Geometry.CircleRadius)
	}
	if f.Geometry.BBox != nil {
		fmt.Fprintf(h, "|b%f,%f,%f,%f", f.Geometry.BBox.MinLat,
			f.Geometry.BBox.MinLon, f.Geometry.BBox.MaxLat, f.Geometry.BBox.MaxLon)
	}

	return h.Sum(nil), nil
}

// ApplyDelta applies a delta to a collection of fences.
func ApplyDelta(existing []FenceItem, delta FenceDelta) ([]FenceItem, error) {
	// Create a map of existing fences for efficient lookup
	fenceMap := make(map[string]FenceItem)
	for _, f := range existing {
		fenceMap[f.ID] = f
	}

	// Remove fences
	for _, id := range delta.RemovedIDs {
		delete(fenceMap, id)
	}

	// Add new fences
	for _, f := range delta.Added {
		if _, exists := fenceMap[f.ID]; exists {
			return nil, fmt.Errorf("fence %s already exists", f.ID)
		}
		fenceMap[f.ID] = f
	}

	// Update existing fences
	for _, f := range delta.Updated {
		if _, exists := fenceMap[f.ID]; !exists {
			return nil, fmt.Errorf("fence %s does not exist", f.ID)
		}
		fenceMap[f.ID] = f
	}

	// Convert back to slice
	result := make([]FenceItem, 0, len(fenceMap))
	for _, f := range fenceMap {
		result = append(result, f)
	}

	return result, nil
}

// CreateDelta computes the delta between two fence collections.
func CreateDelta(oldFences, newFences []FenceItem) FenceDelta {
	oldMap := make(map[string]FenceItem)
	newMap := make(map[string]FenceItem)

	for _, f := range oldFences {
		oldMap[f.ID] = f
	}
	for _, f := range newFences {
		newMap[f.ID] = f
	}

	delta := FenceDelta{}

	// Find removed and updated
	for id, oldF := range oldMap {
		newF, exists := newMap[id]
		if !exists {
			delta.RemovedIDs = append(delta.RemovedIDs, id)
		} else {
			// Check if updated (simple comparison)
			if !fencesEqual(oldF, newF) {
				delta.Updated = append(delta.Updated, newF)
			}
		}
	}

	// Find added
	for id, newF := range newMap {
		if _, exists := oldMap[id]; !exists {
			delta.Added = append(delta.Added, newF)
		}
	}

	return delta
}

func fencesEqual(a, b FenceItem) bool {
	return a.ID == b.ID &&
		a.Type == b.Type &&
		a.StartTS == b.StartTS &&
		a.EndTS == b.EndTS &&
		a.Priority == b.Priority &&
		a.MaxAltitude == b.MaxAltitude &&
		a.MaxSpeed == b.MaxSpeed &&
		a.Name == b.Name &&
		a.Description == b.Description
}
