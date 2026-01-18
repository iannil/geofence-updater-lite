// Package geofence provides core data structures and operations for
// managing geofence restrictions in the GUL system.
package geofence

import (
	"fmt"
	"time"
)

// FenceType defines the category of the geofence restriction.
type FenceType int32

const (
	FenceTypeUnknown          FenceType = 0
	FenceTypeTempRestriction  FenceType = 1 // Temporary restriction for events, emergencies
	FenceTypePermanentNoFly   FenceType = 2 // Permanent no-fly zone
	FenceTypeAltitudeLimit    FenceType = 3 // Maximum altitude restriction
	FenceTypeAltitudeMinimum  FenceType = 4 // Minimum altitude requirement
	FenceTypeSpeedLimit       FenceType = 5 // Speed restriction zone
)

// String returns a human-readable representation of the fence type.
func (t FenceType) String() string {
	switch t {
	case FenceTypeTempRestriction:
		return "TEMP_RESTRICTION"
	case FenceTypePermanentNoFly:
		return "PERMANENT_NO_FLY"
	case FenceTypeAltitudeLimit:
		return "ALTITUDE_LIMIT"
	case FenceTypeAltitudeMinimum:
		return "ALTITUDE_MINIMUM"
	case FenceTypeSpeedLimit:
		return "SPEED_LIMIT"
	default:
		return "UNKNOWN"
	}
}

// Point represents a single WGS84 coordinate.
type Point struct {
	Latitude  float64 `json:"lat"`  // Degrees, -90 to 90
	Longitude float64 `json:"lon"`  // Degrees, -180 to 180
}

// NewPoint creates a new Point with validation.
func NewPoint(lat, lon float64) (Point, error) {
	if lat < -90 || lat > 90 {
		return Point{}, fmt.Errorf("latitude out of range: %f", lat)
	}
	if lon < -180 || lon > 180 {
		return Point{}, fmt.Errorf("longitude out of range: %f", lon)
	}
	return Point{Latitude: lat, Longitude: lon}, nil
}

// Geometry defines the spatial shape of a fence.
type Geometry struct {
	// Polygon vertices (if shape is a polygon)
	Polygon []Point `json:"polygon,omitempty"`

	// Circle center and radius (if shape is a circle)
	CircleCenter *Point  `json:"circle_center,omitempty"`
	CircleRadius float64 `json:"circle_radius_m,omitempty"` // meters

	// Bounding box (if shape is a rectangle)
	BBox *BoundingBox `json:"bbox,omitempty"`
}

// BoundingBox represents a rectangular area.
type BoundingBox struct {
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
}

// Contains checks if a point is within the bounding box.
func (b *BoundingBox) Contains(p Point) bool {
	return p.Latitude >= b.MinLat && p.Latitude <= b.MaxLat &&
		p.Longitude >= b.MinLon && p.Longitude <= b.MaxLon
}

// FenceItem represents a single geofence restriction.
// This is the core data unit that gets signed and distributed.
type FenceItem struct {
	ID          string    `json:"id"`
	Type        FenceType `json:"type"`
	Geometry    Geometry  `json:"geometry"`
	StartTS     int64     `json:"start_ts"`     // Unix timestamp in seconds
	EndTS       int64     `json:"end_ts"`       // Unix timestamp, 0 = no expiry
	Priority    uint32    `json:"priority"`     // Higher = more important
	MaxAltitude uint32    `json:"max_alt_m"`    // Max altitude in meters, 0 = no limit
	MaxSpeed    uint32    `json:"max_speed_mps"` // Max speed in m/s, 0 = no limit
	Name        string    `json:"name"`
	Description string    `json:"description"`

	// Signature fields
	Signature []byte `json:"signature"` // Ed25519 signature
	KeyID     string `json:"key_id"`    // Public key ID
}

// IsActiveAt checks if the fence is active at the given time.
func (f *FenceItem) IsActiveAt(t time.Time) bool {
	ts := t.Unix()
	if ts < f.StartTS {
		return false
	}
	if f.EndTS > 0 && ts > f.EndTS {
		return false
	}
	return true
}

// IsActiveNow checks if the fence is currently active.
func (f *FenceItem) IsActiveNow() bool {
	return f.IsActiveAt(time.Now())
}

// GetBounds returns the bounding box of this fence's geometry.
func (f *FenceItem) GetBounds() BoundingBox {
	if b := f.Geometry.BBox; b != nil {
		return *b
	}
	if len(f.Geometry.Polygon) > 0 {
		return boundsFromPoints(f.Geometry.Polygon)
	}
	if f.Geometry.CircleCenter != nil {
		// Approximate bounds from circle
		const approxLatDeg = 111000 // meters per degree latitude
		latDelta := (f.Geometry.CircleRadius / approxLatDeg)
		lonDelta := (f.Geometry.CircleRadius / approxLatDeg) /
			cosDegrees(f.Geometry.CircleCenter.Latitude)
		return BoundingBox{
			MinLat: f.Geometry.CircleCenter.Latitude - latDelta,
			MaxLat: f.Geometry.CircleCenter.Latitude + latDelta,
			MinLon: f.Geometry.CircleCenter.Longitude - lonDelta,
			MaxLon: f.Geometry.CircleCenter.Longitude + lonDelta,
		}
	}
	return BoundingBox{}
}

func boundsFromPoints(points []Point) BoundingBox {
	if len(points) == 0 {
		return BoundingBox{}
	}
	b := BoundingBox{
		MinLat: points[0].Latitude,
		MaxLat: points[0].Latitude,
		MinLon: points[0].Longitude,
		MaxLon: points[0].Longitude,
	}
	for _, p := range points[1:] {
		if p.Latitude < b.MinLat {
			b.MinLat = p.Latitude
		}
		if p.Latitude > b.MaxLat {
			b.MaxLat = p.Latitude
		}
		if p.Longitude < b.MinLon {
			b.MinLon = p.Longitude
		}
		if p.Longitude > b.MaxLon {
			b.MaxLon = p.Longitude
		}
	}
	return b
}

func cosDegrees(deg float64) float64 {
	// Approximate cos in degrees
	const radPerDeg = 0.017453292519943295
	rad := deg * radPerDeg
	// Small angle approximation with correction
	x := rad * rad
	return 1 - x/2 + x*x/24
}

// FenceCollection represents a batch of fence items.
type FenceCollection struct {
	Items     []FenceItem `json:"items"`
	CreatedTS int64       `json:"created_ts"`
	Version   string      `json:"version"`
}

// FenceDelta represents changes between two versions.
type FenceDelta struct {
	Added      []FenceItem `json:"added"`
	RemovedIDs []string    `json:"removed_ids"`
	Updated    []FenceItem `json:"updated"`
}

// Manifest represents the current state of the geofence database.
type Manifest struct {
	Version        uint64 `json:"version"`
	Timestamp      int64  `json:"timestamp"`
	RootHash       []byte `json:"root_hash"`
	DeltaURL       string `json:"delta_url"`
	SnapshotURL    string `json:"snapshot_url"`
	DeltaSize      uint64 `json:"delta_size"`
	SnapshotSize   uint64 `json:"snapshot_size"`
	DeltaHash      []byte `json:"delta_hash"`
	SnapshotHash   []byte `json:"snapshot_hash"`
	MinClientV     uint32 `json:"min_client_version"`
	Message        string `json:"message"`
	Signature      []byte `json:"signature"`
	KeyID          string `json:"key_id"`
}

// CheckResult represents the result of checking if a location is allowed.
type CheckResult struct {
	Allowed      bool        `json:"allowed"`
	Restriction  *FenceItem  `json:"restriction,omitempty"`
	MatchingFences []FenceItem `json:"matching_fences"`
}

// UpdaterConfig is the configuration for the geofence updater.
type UpdaterConfig struct {
	ManifestURL string `json:"manifest_url"`
	PublicKey   []byte `json:"public_key"`
	StorePath   string `json:"store_path"`

	// Sync interval, defaults to 1 minute
	SyncInterval time.Duration `json:"sync_interval"`

	// HTTP timeout for downloads
	HTTPTimeout time.Duration `json:"http_timeout"`

	// Maximum size of data to download
	MaxDownloadSize int64 `json:"max_download_size"`
}

// Validate checks if the configuration is valid.
func (c *UpdaterConfig) Validate() error {
	if c.ManifestURL == "" {
		return fmt.Errorf("manifest_url is required")
	}
	if len(c.PublicKey) == 0 {
		return fmt.Errorf("public_key is required")
	}
	if c.StorePath == "" {
		return fmt.Errorf("store_path is required")
	}
	if c.SyncInterval == 0 {
		c.SyncInterval = time.Minute
	}
	if c.HTTPTimeout == 0 {
		c.HTTPTimeout = 30 * time.Second
	}
	if c.MaxDownloadSize == 0 {
		c.MaxDownloadSize = 100 * 1024 * 1024 // 100 MB
	}
	return nil
}
