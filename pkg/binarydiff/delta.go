// Package binarydiff provides binary diff operations for efficient
// delta generation between fence database versions.
package binarydiff

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

// DeltaFile represents a binary diff file.
type DeltaFile struct {
	FromVersion uint64
	ToVersion   uint64
	FromSize    int64
	ToSize      int64
	DiffData    []byte
	DiffHash    []byte // SHA-256 of diff data
}

// Diff creates a binary diff between two fence collections.
func Diff(oldFences, newFences []geofence.FenceItem) (*DeltaFile, error) {
	// Serialize old and new collections to JSON
	oldData, err := json.Marshal(geofence.FenceCollection{Items: oldFences})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal old fences: %w", err)
	}

	newData, err := json.Marshal(geofence.FenceCollection{Items: newFences})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new fences: %w", err)
	}

	// Compute binary diff
	diffData, err := computeDiff(oldData, newData)
	if err != nil {
		return nil, fmt.Errorf("failed to compute binary diff: %w", err)
	}

	// Compute hash of diff data
	diffHash := sha256.Sum256(diffData)

	return &DeltaFile{
		FromVersion: 0, // Set by caller
		ToVersion:   0, // Set by caller
		FromSize:    int64(len(oldData)),
		ToSize:      int64(len(newData)),
		DiffData:    diffData,
		DiffHash:    diffHash[:],
	}, nil
}

// computeDiff computes a simple binary diff between two byte slices.
// For a production system, this would use bsdiff for better compression.
func computeDiff(oldData, newData []byte) ([]byte, error) {
	// For now, use a simple diff format:
	// - If data is similar (small change), encode as operations
	// - Otherwise, just return the new data

	// Simple heuristic: if size difference is small, use operations
	const threshold = 0.5 // If new data is >50% different, just return it

	if len(newData) < len(oldData) {
		// New data is smaller, just return it
		return newData, nil
	}

	if float64(len(newData)-len(oldData))/float64(len(oldData)) > threshold {
		// Too different, just return new data
		return newData, nil
	}

	// For similar data, compute a simple delta
	// Find common prefix and suffix
	prefixLen := commonPrefixLen(oldData, newData)
	suffixLen := commonSuffixLen(oldData[prefixLen:], newData[prefixLen:])

	// Build delta: [prefixLen:4][suffixLen:4][middle_data]
	var delta bytes.Buffer
	binary.Write(&delta, binary.LittleEndian, uint32(prefixLen))
	binary.Write(&delta, binary.LittleEndian, uint32(suffixLen))

	// Write the changed middle portion
	middleStart := prefixLen
	middleEnd := len(newData) - suffixLen
	delta.Write(newData[middleStart:middleEnd])

	return delta.Bytes(), nil
}

// commonPrefixLen returns the length of the common prefix of two byte slices.
func commonPrefixLen(a, b []byte) int {
	maxLen := len(a)
	if len(b) < maxLen {
		maxLen = len(b)
	}

	for i := 0; i < maxLen; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return maxLen
}

// commonSuffixLen returns the length of the common suffix of two byte slices.
func commonSuffixLen(a, b []byte) int {
	maxLen := len(a)
	if len(b) < maxLen {
		maxLen = len(b)
	}

	for i := 0; i < maxLen; i++ {
		ai := len(a) - 1 - i
		bi := len(b) - 1 - i
		if ai < 0 || bi < 0 || a[ai] != b[bi] {
			return i
		}
	}
	return maxLen
}

// Patch applies a binary diff to old data to produce new data.
func Patch(oldData []byte, diffData []byte) ([]byte, error) {
	// Check if this is our delta format or just raw data
	if len(diffData) < 8 {
		// Too small for delta format, assume it's raw data
		return diffData, nil
	}

	// Read prefix and suffix lengths
	prefixLen := binary.LittleEndian.Uint32(diffData[0:4])
	suffixLen := binary.LittleEndian.Uint32(diffData[4:8])

	// Validate lengths
	if prefixLen > uint32(len(oldData)) || suffixLen > uint32(len(oldData)) {
		// Invalid delta, return as-is
		return diffData, nil
	}

	// Apply patch
	totalLen := int(prefixLen) + len(diffData[8:]) + int(suffixLen)
	if totalLen > 10*1024*1024 { // Sanity check: max 10MB
		return diffData, nil
	}

	// Build result: prefix + new_middle + suffix
	result := make([]byte, 0, totalLen)
	result = append(result, oldData[:prefixLen]...)
	result = append(result, diffData[8:]...)
	if suffixLen > 0 && int(prefixLen)+int(suffixLen) <= len(oldData) {
		result = append(result, oldData[len(oldData)-int(suffixLen):]...)
	}

	return result, nil
}

// PatchFences applies a binary diff to old fences to produce new fences.
func PatchFences(oldFences []geofence.FenceItem, delta *DeltaFile) ([]geofence.FenceItem, error) {
	// Serialize old fences
	oldData, err := json.Marshal(geofence.FenceCollection{Items: oldFences})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal old fences: %w", err)
	}

	// Verify diff hash
	if len(delta.DiffHash) > 0 {
		h := sha256.Sum256(delta.DiffData)
		if !bytes.Equal(h[:], delta.DiffHash) {
			return nil, fmt.Errorf("diff hash mismatch")
		}
	}

	// Apply patch
	newData, err := Patch(oldData, delta.DiffData)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patch: %w", err)
	}

	// Unmarshal new fences
	var collection geofence.FenceCollection
	if err := json.Unmarshal(newData, &collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal new fences: %w", err)
	}

	return collection.Items, nil
}

// ComputeDeltaSize estimates the size of a delta between two fence collections.
func ComputeDeltaSize(oldFences, newFences []geofence.FenceItem) (int, error) {
	added, removed, updated := computeDeltaStats(oldFences, newFences)

	// Rough estimation: each fence change costs ~100 bytes of metadata
	// plus the size of changed fence data
	estimatedSize := added*200 + updated*300 + removed*50

	return estimatedSize, nil
}

// computeDeltaStats computes the number of added, removed, and updated fences.
func computeDeltaStats(oldFences, newFences []geofence.FenceItem) (added, removed, updated int) {
	oldMap := make(map[string]struct{})
	for _, f := range oldFences {
		oldMap[f.ID] = struct{}{}
	}

	newMap := make(map[string]geofence.FenceItem)
	for _, f := range newFences {
		newMap[f.ID] = f
	}

	// Count removed
	removed = len(oldMap) - len(newMap)
	if removed < 0 {
		removed = 0
	}

	// Count added and updated
	for id := range newMap {
		if _, exists := oldMap[id]; !exists {
			added++
		} else {
			// Assume updated if it exists (simplified check)
			updated++
		}
	}

	return added, removed, updated
}

// WriteDelta writes a delta file to the given writer.
func WriteDelta(delta *DeltaFile, w io.Writer) error {
	if err := json.NewEncoder(w).Encode(delta); err != nil {
		return fmt.Errorf("failed to write delta: %w", err)
	}
	return nil
}

// ReadDelta reads a delta file from the given reader.
func ReadDelta(r io.Reader) (*DeltaFile, error) {
	var delta DeltaFile
	if err := json.NewDecoder(r).Decode(&delta); err != nil {
		return nil, fmt.Errorf("failed to read delta: %w", err)
	}
	return &delta, nil
}

// DeltaHeader represents the header of a delta file.
type DeltaHeader struct {
	Magic       [4]byte // "GULD"
	Version     uint16  // Protocol version
	FromVersion uint64
	ToVersion   uint64
	OldSize     uint64
	NewSize     uint64
	DiffSize    uint64
	DiffHash    []byte // SHA-256 of diff data
}

const deltaMagic = "GULD"

// WriteDeltaFile writes a complete delta file with header.
func WriteDeltaFile(oldFences, newFences []geofence.FenceItem, fromVer, toVer uint64, w io.Writer) error {
	// Compute delta
	delta, err := Diff(oldFences, newFences)
	if err != nil {
		return fmt.Errorf("failed to create diff: %w", err)
	}

	// Fill header
	delta.FromVersion = fromVer
	delta.ToVersion = toVer

	// Write header
	header := DeltaHeader{
		Magic:       [4]byte{'G', 'U', 'L', 'D'},
		Version:     1,
		FromVersion: fromVer,
		ToVersion:   toVer,
		OldSize:     uint64(delta.FromSize),
		NewSize:     uint64(delta.ToSize),
		DiffSize:    uint64(len(delta.DiffData)),
		DiffHash:    delta.DiffHash,
	}

	if err := binaryWriteHeader(w, &header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write diff data
	if _, err := w.Write(delta.DiffData); err != nil {
		return fmt.Errorf("failed to write diff data: %w", err)
	}

	return nil
}

// ReadDeltaFile reads a complete delta file.
func ReadDeltaFile(r io.Reader, expectedToVer uint64) (*DeltaFile, error) {
	var header DeltaHeader
	if err := binaryReadHeader(r, &header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Validate magic number
	if string(header.Magic[:]) != deltaMagic {
		return nil, fmt.Errorf("invalid magic number: %s", string(header.Magic[:]))
	}

	// Validate version
	if header.Version != 1 {
		return nil, fmt.Errorf("unsupported delta version: %d", header.Version)
	}

	// Validate target version
	if header.ToVersion != expectedToVer {
		return nil, fmt.Errorf("version mismatch: expected %d, got %d", expectedToVer, header.ToVersion)
	}

	// Read diff data
	diffData := make([]byte, header.DiffSize)
	if _, err := io.ReadFull(r, diffData); err != nil {
		return nil, fmt.Errorf("failed to read diff data: %w", err)
	}

	// Verify hash
	if len(header.DiffHash) == sha256.Size {
		h := sha256.Sum256(diffData)
		if !bytes.Equal(h[:], header.DiffHash) {
			return nil, fmt.Errorf("diff hash mismatch")
		}
	}

	return &DeltaFile{
		FromVersion: header.FromVersion,
		ToVersion:   header.ToVersion,
		FromSize:    int64(header.OldSize),
		ToSize:      int64(header.NewSize),
		DiffData:    diffData,
		DiffHash:    header.DiffHash,
	}, nil
}

// binaryWriteHeader writes a delta header in binary format.
func binaryWriteHeader(w io.Writer, h *DeltaHeader) error {
	// Write magic
	if _, err := w.Write(h.Magic[:]); err != nil {
		return err
	}

	// Write version
	if err := binary.Write(w, binary.LittleEndian, h.Version); err != nil {
		return err
	}

	// Write from version
	if err := binary.Write(w, binary.LittleEndian, h.FromVersion); err != nil {
		return err
	}

	// Write to version
	if err := binary.Write(w, binary.LittleEndian, h.ToVersion); err != nil {
		return err
	}

	// Write old size
	if err := binary.Write(w, binary.LittleEndian, h.OldSize); err != nil {
		return err
	}

	// Write new size
	if err := binary.Write(w, binary.LittleEndian, h.NewSize); err != nil {
		return err
	}

	// Write diff size
	if err := binary.Write(w, binary.LittleEndian, h.DiffSize); err != nil {
		return err
	}

	// Write hash
	if len(h.DiffHash) > 0 {
		if _, err := w.Write(h.DiffHash); err != nil {
			return err
		}
	}

	return nil
}

// binaryReadHeader reads a delta header from binary format.
func binaryReadHeader(r io.Reader, h *DeltaHeader) error {
	// Read magic
	if _, err := io.ReadFull(r, h.Magic[:]); err != nil {
		return err
	}

	// Validate magic
	if string(h.Magic[:]) != deltaMagic {
		return fmt.Errorf("invalid magic number: %s", string(h.Magic[:]))
	}

	// Read version
	if err := binary.Read(r, binary.LittleEndian, &h.Version); err != nil {
		return err
	}

	// Read from version
	if err := binary.Read(r, binary.LittleEndian, &h.FromVersion); err != nil {
		return err
	}

	// Read to version
	if err := binary.Read(r, binary.LittleEndian, &h.ToVersion); err != nil {
		return err
	}

	// Read old size
	if err := binary.Read(r, binary.LittleEndian, &h.OldSize); err != nil {
		return err
	}

	// Read new size
	if err := binary.Read(r, binary.LittleEndian, &h.NewSize); err != nil {
		return err
	}

	// Read diff size
	if err := binary.Read(r, binary.LittleEndian, &h.DiffSize); err != nil {
		return err
	}

	// Read hash - read up to sha256.Size bytes
	hashBuf := make([]byte, sha256.Size)
	n, err := io.ReadFull(r, hashBuf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return err
	}
	if n > 0 {
		h.DiffHash = hashBuf[:n]
	}

	return nil
}
