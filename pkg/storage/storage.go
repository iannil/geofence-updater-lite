// Package storage provides persistent storage for geofence data using SQLite
// with R-Tree spatial indexing.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/iannil/geofence-updater-lite/pkg/geofence"
	_ "modernc.org/sqlite"
)

// ErrFenceNotFound is returned when a fence is not found in the store.
var ErrFenceNotFound = errors.New("fence not found")

// Store is the interface for geofence data persistence.
type Store interface {
	// Fence operations
	AddFence(ctx context.Context, fence *geofence.FenceItem) error
	GetFence(ctx context.Context, id string) (*geofence.FenceItem, error)
	UpdateFence(ctx context.Context, fence *geofence.FenceItem) error
	DeleteFence(ctx context.Context, id string) error
	ListFences(ctx context.Context) ([]*geofence.FenceItem, error)

	// Spatial query
	QueryAtPoint(ctx context.Context, lat, lon float64) ([]*geofence.FenceItem, error)
	QueryInBounds(ctx context.Context, bounds *geofence.BoundingBox) ([]*geofence.FenceItem, error)

	// Manifest operations
	GetManifest(ctx context.Context) (*geofence.Manifest, error)
	SetManifest(ctx context.Context, manifest *geofence.Manifest) error

	// Version management
	GetVersion(ctx context.Context) (uint64, error)
	SetVersion(ctx context.Context, version uint64) error

	// Batch operations
	BeginTx(ctx context.Context) (*Tx, error)
	Close() error
}

// Tx represents a transaction.
type Tx struct {
	tx *sql.Tx
}

// Commit commits the transaction.
func (t *Tx) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction.
func (t *Tx) Rollback() error {
	return t.tx.Rollback()
}

// SQLiteStore implements Store using SQLite with R-Tree extension.
type SQLiteStore struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// Config holds configuration for the SQLite store.
type Config struct {
	Path string // Path to the SQLite database file
}

// Open creates or opens an SQLite database for geofence storage.
func Open(ctx context.Context, cfg *Config) (*SQLiteStore, error) {
	if cfg == nil || cfg.Path == "" {
		return nil, fmt.Errorf("config path is required")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	store := &SQLiteStore{
		db:   db,
		path: cfg.Path,
	}

	if err := store.init(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// init creates the necessary tables and indexes.
func (s *SQLiteStore) init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create fences table with integer primary key for R-Tree compatibility
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS fences (
			rowid INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT UNIQUE NOT NULL,
			type INTEGER NOT NULL,
			start_ts INTEGER NOT NULL DEFAULT 0,
			end_ts INTEGER NOT NULL DEFAULT 0,
			priority INTEGER NOT NULL DEFAULT 0,
			max_altitude INTEGER NOT NULL DEFAULT 0,
			max_speed INTEGER NOT NULL DEFAULT 0,
			name TEXT,
			description TEXT,
			signature BLOB,
			key_id TEXT,
			geometry_json TEXT NOT NULL,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create fences table: %w", err)
	}

	// Create R-Tree virtual table for spatial indexing
	_, err = s.db.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS fence_index USING rtree(
			rowid,          -- Links to fences.rowid
			minX, maxX,      -- Longitude bounds
			minY, maxY       -- Latitude bounds
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create rtree table: %w", err)
	}

	// Create metadata table for manifest and version
	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value BLOB NOT NULL,
			updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}

	// Create indexes
	_, err = s.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS fences_type_idx ON fences(type);
		CREATE INDEX IF NOT EXISTS fences_priority_idx ON fences(priority);
		CREATE INDEX IF NOT EXISTS fences_time_idx ON fences(start_ts, end_ts);
		CREATE INDEX IF NOT EXISTS fences_id_idx ON fences(id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// AddFence adds a new fence to the store.
func (s *SQLiteStore) AddFence(ctx context.Context, fence *geofence.FenceItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize geometry to JSON
	geomJSON, err := json.Marshal(fence.Geometry)
	if err != nil {
		return fmt.Errorf("failed to marshal geometry: %w", err)
	}

	// Calculate bounds for R-Tree
	bounds := fence.GetBounds()

	// Use transaction for atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	// Insert fence and get the rowid
	result, err := tx.ExecContext(ctx, `
		INSERT INTO fences (id, type, start_ts, end_ts, priority, max_altitude, max_speed,
			name, description, signature, key_id, geometry_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, fence.ID, int(fence.Type), fence.StartTS, fence.EndTS, fence.Priority,
		fence.MaxAltitude, fence.MaxSpeed, fence.Name, fence.Description,
		fence.Signature, fence.KeyID, string(geomJSON))
	if err != nil {
		return fmt.Errorf("failed to insert fence: %w", err)
	}

	// Get the rowid of the inserted fence
	rowID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get rowid: %w", err)
	}

	// Add to R-Tree using rowid (same transaction)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO fence_index (rowid, minX, maxX, minY, maxY)
		VALUES (?, ?, ?, ?, ?)
	`, rowID, bounds.MinLon, bounds.MaxLon, bounds.MinLat, bounds.MaxLat)
	if err != nil {
		return fmt.Errorf("failed to insert into rtree: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}

// GetFence retrieves a fence by ID.
func (s *SQLiteStore) GetFence(ctx context.Context, id string) (*geofence.FenceItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var fence geofence.FenceItem
	var geomJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, start_ts, end_ts, priority, max_altitude, max_speed,
			name, description, signature, key_id, geometry_json
		FROM fences WHERE id = ?
	`, id).Scan(
		&fence.ID, &fence.Type, &fence.StartTS, &fence.EndTS, &fence.Priority,
		&fence.MaxAltitude, &fence.MaxSpeed, &fence.Name, &fence.Description,
		&fence.Signature, &fence.KeyID, &geomJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fence not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query fence: %w", err)
	}

	if err := json.Unmarshal([]byte(geomJSON), &fence.Geometry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal geometry: %w", err)
	}

	return &fence, nil
}

// UpdateFence updates an existing fence.
func (s *SQLiteStore) UpdateFence(ctx context.Context, fence *geofence.FenceItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize geometry
	geomJSON, err := json.Marshal(fence.Geometry)
	if err != nil {
		return fmt.Errorf("failed to marshal geometry: %w", err)
	}

	// Calculate bounds
	bounds := fence.GetBounds()

	// Use transaction for atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	// Update fence
	result, err := tx.ExecContext(ctx, `
		UPDATE fences SET type = ?, start_ts = ?, end_ts = ?, priority = ?,
			max_altitude = ?, max_speed = ?, name = ?, description = ?,
			signature = ?, key_id = ?, geometry_json = ?, updated_at = strftime('%s', 'now')
		WHERE id = ?
	`, int(fence.Type), fence.StartTS, fence.EndTS, fence.Priority,
		fence.MaxAltitude, fence.MaxSpeed, fence.Name, fence.Description,
		fence.Signature, fence.KeyID, string(geomJSON), fence.ID)
	if err != nil {
		return fmt.Errorf("failed to update fence: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrFenceNotFound
	}

	// Update R-Tree using rowid (same transaction)
	_, err = tx.ExecContext(ctx, `
		UPDATE fence_index SET minX = ?, maxX = ?, minY = ?, maxY = ? WHERE rowid = (SELECT rowid FROM fences WHERE id = ?)
	`, bounds.MinLon, bounds.MaxLon, bounds.MinLat, bounds.MaxLat, fence.ID)
	if err != nil {
		return fmt.Errorf("failed to update rtree: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}

// DeleteFence removes a fence from the store.
func (s *SQLiteStore) DeleteFence(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use transaction for atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	// Get rowid first
	var rowID int64
	err = tx.QueryRowContext(ctx, "SELECT rowid FROM fences WHERE id = ?", id).Scan(&rowID)
	if err == sql.ErrNoRows {
		return ErrFenceNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to get rowid: %w", err)
	}

	// Delete from fences table
	result, err := tx.ExecContext(ctx, "DELETE FROM fences WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete fence: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrFenceNotFound
	}

	// Delete from R-Tree (same transaction)
	_, err = tx.ExecContext(ctx, "DELETE FROM fence_index WHERE rowid = ?", rowID)
	if err != nil {
		return fmt.Errorf("failed to delete from rtree: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}

// ListFences returns all fences.
func (s *SQLiteStore) ListFences(ctx context.Context) ([]*geofence.FenceItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, type, start_ts, end_ts, priority, max_altitude, max_speed,
			name, description, signature, key_id, geometry_json
		FROM fences ORDER BY priority DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query fences: %w", err)
	}
	defer rows.Close()

	var fences []*geofence.FenceItem
	for rows.Next() {
		var fence geofence.FenceItem
		var geomJSON string

		err := rows.Scan(
			&fence.ID, &fence.Type, &fence.StartTS, &fence.EndTS, &fence.Priority,
			&fence.MaxAltitude, &fence.MaxSpeed, &fence.Name, &fence.Description,
			&fence.Signature, &fence.KeyID, &geomJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fence: %w", err)
		}

		if err := json.Unmarshal([]byte(geomJSON), &fence.Geometry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal geometry: %w", err)
		}

		fences = append(fences, &fence)
	}

	return fences, rows.Err()
}

// QueryAtPoint finds all fences containing the given point.
func (s *SQLiteStore) QueryAtPoint(ctx context.Context, lat, lon float64) ([]*geofence.FenceItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Use R-Tree to find candidate fences
	// Note: R-Tree gives us bounding box matches, we still need to check exact geometry
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id, f.type, f.start_ts, f.end_ts, f.priority, f.max_altitude, f.max_speed,
			f.name, f.description, f.signature, f.key_id, f.geometry_json
		FROM fences f
		INNER JOIN fence_index idx ON f.rowid = idx.rowid
		WHERE idx.minX <= ? AND idx.maxX >= ? AND idx.minY <= ? AND idx.maxY >= ?
		ORDER BY f.priority DESC
	`, lon, lon, lat, lat)
	if err != nil {
		return nil, fmt.Errorf("failed to query rtree: %w", err)
	}
	defer rows.Close()

	var fences []*geofence.FenceItem
	point := geofence.Point{Latitude: lat, Longitude: lon}

	for rows.Next() {
		var fence geofence.FenceItem
		var geomJSON string

		err := rows.Scan(
			&fence.ID, &fence.Type, &fence.StartTS, &fence.EndTS, &fence.Priority,
			&fence.MaxAltitude, &fence.MaxSpeed, &fence.Name, &fence.Description,
			&fence.Signature, &fence.KeyID, &geomJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fence: %w", err)
		}

		if err := json.Unmarshal([]byte(geomJSON), &fence.Geometry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal geometry: %w", err)
		}

		// Check exact geometry match
		if fence.ContainsPoint(point) && fence.IsActiveNow() {
			fences = append(fences, &fence)
		}
	}

	return fences, rows.Err()
}

// QueryInBounds finds all fences intersecting the given bounding box.
func (s *SQLiteStore) QueryInBounds(ctx context.Context, bounds *geofence.BoundingBox) ([]*geofence.FenceItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id, f.type, f.start_ts, f.end_ts, f.priority, f.max_altitude, f.max_speed,
			f.name, f.description, f.signature, f.key_id, f.geometry_json
		FROM fences f
		INNER JOIN fence_index idx ON f.rowid = idx.rowid
		WHERE idx.maxX >= ? AND idx.minX <= ? AND idx.maxY >= ? AND idx.minY <= ?
		ORDER BY f.priority DESC
	`, bounds.MinLon, bounds.MaxLon, bounds.MaxLat, bounds.MinLat)
	if err != nil {
		return nil, fmt.Errorf("failed to query rtree: %w", err)
	}
	defer rows.Close()

	var fences []*geofence.FenceItem
	for rows.Next() {
		var fence geofence.FenceItem
		var geomJSON string

		err := rows.Scan(
			&fence.ID, &fence.Type, &fence.StartTS, &fence.EndTS, &fence.Priority,
			&fence.MaxAltitude, &fence.MaxSpeed, &fence.Name, &fence.Description,
			&fence.Signature, &fence.KeyID, &geomJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fence: %w", err)
		}

		if err := json.Unmarshal([]byte(geomJSON), &fence.Geometry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal geometry: %w", err)
		}

		fences = append(fences, &fence)
	}

	return fences, rows.Err()
}

// GetManifest retrieves the stored manifest.
func (s *SQLiteStore) GetManifest(ctx context.Context) (*geofence.Manifest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data []byte
	err := s.db.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key = 'manifest'").Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil // No manifest stored yet
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query manifest: %w", err)
	}

	var manifest geofence.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return &manifest, nil
}

// SetManifest stores a manifest.
func (s *SQLiteStore) SetManifest(ctx context.Context, manifest *geofence.Manifest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO metadata (key, value, updated_at)
		VALUES ('manifest', ?, strftime('%s', 'now'))
	`, data)
	if err != nil {
		return fmt.Errorf("failed to store manifest: %w", err)
	}

	return nil
}

// GetVersion retrieves the current version.
func (s *SQLiteStore) GetVersion(ctx context.Context) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var version uint64
	err := s.db.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key = 'version'").Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil // No version set yet
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query version: %w", err)
	}

	return version, nil
}

// SetVersion stores the current version.
func (s *SQLiteStore) SetVersion(ctx context.Context, version uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO metadata (key, value, updated_at)
		VALUES ('version', ?, strftime('%s', 'now'))
	`, version)
	if err != nil {
		return fmt.Errorf("failed to store version: %w", err)
	}

	return nil
}

// BeginTx starts a new transaction.
func (s *SQLiteStore) BeginTx(ctx context.Context) (*Tx, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &Tx{tx: tx}, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}
