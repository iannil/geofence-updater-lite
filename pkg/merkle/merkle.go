// Package merkle provides Merkle Tree implementation for efficient
// version hashing and delta computation in the GUL system.
package merkle

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

const (
	// HashSize is the size of a SHA-256 hash in bytes.
	HashSize = sha256.Size
)

// Hash represents a SHA-256 hash.
type Hash [HashSize]byte

// String returns the hex representation of the hash.
func (h Hash) String() string {
	return fmt.Sprintf("%x", h[:])
}

// HashFromString creates a Hash from a hex string.
func HashFromString(s string) (Hash, error) {
	var h Hash
	b, err := decodeHex(s)
	if err != nil {
		return h, err
	}
	copy(h[:], b)
	return h, nil
}

func decodeHex(s string) ([]byte, error) {
	b := make([]byte, len(s)/2)
	for i := 0; i < len(b); i++ {
		var x int64
		_, err := fmt.Sscanf(s[i*2:i*2+2], "%02x", &x)
		if err != nil {
			return nil, fmt.Errorf("invalid hex: %w", err)
		}
		b[i] = byte(x)
	}
	return b, nil
}

// Node represents a node in the Merkle tree.
type Node struct {
	Hash     Hash
	Left     *Node
	Right    *Node
	Leaf     bool
	LeafID    string   // ID of the fence item if this is a leaf
	LeafData []byte  // Encoded fence item data
}

// Tree represents a Merkle tree of fence items.
type Tree struct {
	root   *Node
	leaves map[string]*Node // Map from fence ID to leaf node
	mu     sync.RWMutex
}

// NewTree creates a new Merkle tree from a slice of fence items.
func NewTree(fences []geofence.FenceItem) (*Tree, error) {
	t := &Tree{
		leaves: make(map[string]*Node),
	}

	if len(fences) == 0 {
		return t, nil
	}

	// Sort fences by ID for deterministic tree building
	sortedFences := make([]geofence.FenceItem, len(fences))
	copy(sortedFences, fences)
	sort.Slice(sortedFences, func(i, j int) bool {
		return sortedFences[i].ID < sortedFences[j].ID
	})

	// Create leaf nodes
	leaves := make([]*Node, 0, len(sortedFences))
	for _, fence := range sortedFences {
		data, err := json.Marshal(fence)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal fence %s: %w", fence.ID, err)
		}

		// Remove signature for hashing (signature is over different data)
		fenceCopy := fence
		fenceCopy.Signature = nil
		dataForHash, err := json.Marshal(fenceCopy)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal fence for hashing %s: %w", fence.ID, err)
		}

		h := sha256.Sum256(dataForHash)
		node := &Node{
			Hash:     h,
			Leaf:     true,
			LeafID:    fence.ID,
			LeafData: data,
		}
		t.leaves[fence.ID] = node
		leaves = append(leaves, node)
	}

	t.root = t.buildTree(leaves)
	return t, nil
}

// buildTree recursively builds the Merkle tree from leaf nodes.
func (t *Tree) buildTree(nodes []*Node) *Node {
	if len(nodes) == 0 {
		return nil
	}
	if len(nodes) == 1 {
		return nodes[0]
	}

	// Build tree bottom-up
	for len(nodes) > 1 {
		var newLevel []*Node

		// Process pairs
		for i := 0; i < len(nodes); i += 2 {
			left := nodes[i]

			var right *Node
			if i+1 < len(nodes) {
				right = nodes[i+1]
			}

			// Create parent node
			h := sha256.New()
			if left != nil {
				h.Write(left.Hash[:])
			}
			if right != nil {
				h.Write(right.Hash[:])
			}
			parentHash := h.Sum(nil)
			var parentHashArray Hash
			copy(parentHashArray[:], parentHash)

			parent := &Node{
				Hash:  parentHashArray,
				Left:  left,
				Right: right,
			}
			newLevel = append(newLevel, parent)
		}

		nodes = newLevel
	}

	return nodes[0]
}

// RootHash returns the root hash of the Merkle tree.
func (t *Tree) RootHash() Hash {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.root == nil {
		return Hash{}
	}
	return t.root.Hash
}

// GetProof returns a Merkle proof for the given fence ID.
func (t *Tree) GetProof(fenceID string) ([][]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	leaf, exists := t.leaves[fenceID]
	if !exists {
		return nil, fmt.Errorf("fence not found: %s", fenceID)
	}

	var proof [][]byte

	// Start from leaf and go up to root
	current := leaf
	for current != t.root {
		// Find parent and sibling
		parent := t.findParent(current)
		if parent == nil {
			break
		}

		// Add sibling hash to proof
		var siblingHash []byte
		if parent.Left == current {
			if parent.Right != nil {
				siblingHash = parent.Right.Hash[:]
			}
		} else {
			siblingHash = parent.Left.Hash[:]
		}

		if len(siblingHash) > 0 {
			proof = append(proof, siblingHash)
		}

		current = parent
	}

	return proof, nil
}

// findParent finds the parent node of a given node.
func (t *Tree) findParent(child *Node) *Node {
	return t.findParentRecursive(t.root, child)
}

func (t *Tree) findParentRecursive(node, child *Node) *Node {
	if node == nil || node.Leaf {
		return nil
	}

	if node.Left == child || node.Right == child {
		return node
	}

	if parent := t.findParentRecursive(node.Left, child); parent != nil {
		return parent
	}

	return t.findParentRecursive(node.Right, child)
}

// VerifyProof verifies a Merkle proof for a given fence ID and root hash.
func VerifyProof(fenceID string, fenceData []byte, proof [][]byte, rootHash Hash) (bool, error) {
	// Hash the fence data (without signature)
	var fence geofence.FenceItem
	if err := json.Unmarshal(fenceData, &fence); err != nil {
		return false, fmt.Errorf("failed to unmarshal fence: %w", err)
	}
	fence.Signature = nil
	data, err := json.Marshal(fence)
	if err != nil {
		return false, fmt.Errorf("failed to marshal fence: %w", err)
	}

	// Hash the leaf
	h := sha256.Sum256(data)
	currentHash := h

	// Verify proof path
	for _, siblingHash := range proof {
		if len(siblingHash) == 0 {
			continue // Odd number of nodes at this level
		}

		// Combine current hash with sibling hash
		hasher := sha256.New()
		hasher.Write(currentHash[:])
		hasher.Write(siblingHash)
		currentHash = sha256.Sum256(hasher.Sum(nil))
	}

	return currentHash == rootHash, nil
}

// computeDelta computes the added, removed, and updated fences between two collections.
func computeDelta(oldFences, newFences []geofence.FenceItem) (added, updated []geofence.FenceItem, removedIDs []string) {
	oldMap := make(map[string]geofence.FenceItem)
	for _, f := range oldFences {
		oldMap[f.ID] = f
	}

	newMap := make(map[string]geofence.FenceItem)
	for _, f := range newFences {
		newMap[f.ID] = f
	}

	// Find removed fences
	for id := range oldMap {
		if _, exists := newMap[id]; !exists {
			removedIDs = append(removedIDs, id)
		}
	}

	// Find added and updated fences
	for id, newF := range newMap {
		if oldF, exists := oldMap[id]; !exists {
			added = append(added, newF)
		} else if !fencesEqual(oldF, newF) {
			updated = append(updated, newF)
		}
	}

	return added, updated, removedIDs
}

// fencesEqual checks if two fences are equal (excluding signature).
func fencesEqual(a, b geofence.FenceItem) bool {
	// Quick ID check
	if a.ID != b.ID || a.Type != b.Type || a.Priority != b.Priority ||
		a.StartTS != b.StartTS || a.EndTS != b.EndTS ||
		a.MaxAltitude != b.MaxAltitude || a.MaxSpeed != b.MaxSpeed ||
		a.Name != b.Name || a.Description != b.Description ||
		a.KeyID != b.KeyID {
		return false
	}
	// Compare geometry
	aGeom, _ := json.Marshal(a.Geometry)
	bGeom, _ := json.Marshal(b.Geometry)
	return string(aGeom) == string(bGeom)
}

// ComputeDelta computes the delta between two fence collections.
func ComputeDelta(oldFences, newFences []geofence.FenceItem) ([]byte, int64, error) {
	// Build delta struct
	added, updated, removed := computeDelta(oldFences, newFences)

	// For now, we'll create a simple JSON representation
	// In Phase 4 complete, this would use bsdiff for binary delta
	delta := geofence.FenceDelta{
		Added:      added,
		RemovedIDs: removed,
		Updated:    updated,
	}

	data, err := json.Marshal(delta)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal delta: %w", err)
	}

	return data, int64(len(data)), nil
}

// ApplyDelta applies a delta to a collection of fences.
func ApplyDelta(existingFences []geofence.FenceItem, deltaData []byte) ([]geofence.FenceItem, error) {
	var delta geofence.FenceDelta
	if err := json.Unmarshal(deltaData, &delta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal delta: %w", err)
	}

	return geofence.ApplyDelta(existingFences, delta)
}

// VersionInfo contains version metadata for the delta.
type VersionInfo struct {
	Version     uint64
	FromVersion uint64
	FromHash    []byte
	ToHash      []byte
	DeltaSize   int64
	Timestamp   int64
}

// CreateSnapshot creates a snapshot of the current fences.
func CreateSnapshot(fences []geofence.FenceItem) ([]byte, int64, error) {
	collection := geofence.FenceCollection{
		Items:     fences,
		CreatedTS: time.Now().Unix(),
		Version:   "",
	}

	data, err := json.Marshal(collection)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	return data, int64(len(data)), nil
}

// LoadSnapshot loads a snapshot of fences.
func LoadSnapshot(data []byte) ([]geofence.FenceItem, error) {
	var collection geofence.FenceCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return collection.Items, nil
}
