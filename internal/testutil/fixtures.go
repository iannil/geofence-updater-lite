// Package testutil provides test fixtures and utilities for GUL testing.
package testutil

import (
	"time"

	"github.com/iannil/geofence-updater-lite/pkg/geofence"
)

// MustNewPoint creates a Point or panics.
func MustNewPoint(lat, lon float64) geofence.Point {
	p, err := geofence.NewPoint(lat, lon)
	if err != nil {
		panic(err)
	}
	return p
}

// TemporaryFence returns a test temporary restriction fence.
func TemporaryFence() geofence.FenceItem {
	now := time.Now()
	return geofence.FenceItem{
		ID:     "test-temp-001",
		Type:   geofence.FenceTypeTempRestriction,
		StartTS: now.Unix(),
		EndTS:   now.Add(24 * time.Hour).Unix(),
		Priority: 50,
		Name:     "Test Temporary Restriction",
		Description: "Temporary no-fly zone for testing",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				MustNewPoint(39.9042, 116.4074), // Beijing
				MustNewPoint(39.9142, 116.4074),
				MustNewPoint(39.9142, 116.4174),
				MustNewPoint(39.9042, 116.4174),
			},
		},
	}
}

// PermanentNoFlyZone returns a test permanent no-fly zone.
func PermanentNoFlyZone() geofence.FenceItem {
	return geofence.FenceItem{
		ID:     "test-perm-001",
		Type:   geofence.FenceTypePermanentNoFly,
		StartTS: 0,
		EndTS:   0,
		Priority: 100,
		Name:     "Test Airport No-Fly Zone",
		Description: "Permanent restriction around airport",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				MustNewPoint(31.1443, 121.8083), // Shanghai area
				MustNewPoint(31.1543, 121.8083),
				MustNewPoint(31.1543, 121.8183),
				MustNewPoint(31.1443, 121.8183),
			},
		},
	}
}

// AltitudeLimitFence returns a test altitude limit fence.
func AltitudeLimitFence() geofence.FenceItem {
	return geofence.FenceItem{
		ID:     "test-alt-001",
		Type:   geofence.FenceTypeAltitudeLimit,
		StartTS: 0,
		EndTS:   0,
		Priority: 30,
		MaxAltitude: 120, // 120 meters
		Name:     "Test Altitude Limit",
		Description: "Maximum altitude 120m",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				MustNewPoint(22.5431, 114.0579), // Shenzhen
				MustNewPoint(22.5531, 114.0579),
				MustNewPoint(22.5531, 114.0679),
				MustNewPoint(22.5431, 114.0679),
			},
		},
	}
}

// CircleFence returns a test circular fence.
func CircleFence() geofence.FenceItem {
	center := MustNewPoint(39.9042, 116.4074) // Beijing
	return geofence.FenceItem{
		ID:     "test-circle-001",
		Type:   geofence.FenceTypeTempRestriction,
		StartTS: 0,
		EndTS:   0,
		Priority: 60,
		Name:     "Test Circular Restriction",
		Description: "Circular no-fly zone",
		Geometry: geofence.Geometry{
			CircleCenter: &center,
			CircleRadius: 500, // 500 meters
		},
	}
}

// SampleFences returns a slice of test fences.
func SampleFences() []geofence.FenceItem {
	return []geofence.FenceItem{
		TemporaryFence(),
		PermanentNoFlyZone(),
		AltitudeLimitFence(),
		CircleFence(),
	}
}

// BeijingTiananmen returns a fence around Beijing Tiananmen Square.
func BeijingTiananmen() geofence.FenceItem {
	return geofence.FenceItem{
		ID:     "cn-bj-tiananmen",
		Type:   geofence.FenceTypePermanentNoFly,
		StartTS: 0,
		EndTS:   0,
		Priority: 100,
		Name:     "Beijing Tiananmen Square",
		Description: "Permanent no-fly zone over Tiananmen Square",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				MustNewPoint(39.9035, 116.3915),
				MustNewPoint(39.9095, 116.3915),
				MustNewPoint(39.9095, 116.4045),
				MustNewPoint(39.9035, 116.4045),
			},
		},
	}
}

// SampleManifest returns a test manifest.
func SampleManifest() *geofence.Manifest {
	return &geofence.Manifest{
		Version:     1,
		Timestamp:   time.Now().Unix(),
		DeltaURL:    "/patches/v0_to_v1.bin",
		SnapshotURL: "/snapshots/v1.bin",
		DeltaSize:   1024,
		SnapshotSize: 4096,
		Message:     "Initial release",
	}
}
