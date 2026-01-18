package geofence

import (
	"math"
	"testing"
	"time"
)

// Test fixtures (moved from internal/testutil to avoid import cycle)

func testPoint(lat, lon float64) Point {
	p, err := NewPoint(lat, lon)
	if err != nil {
		panic(err)
	}
	return p
}

func testTemporaryFence() FenceItem {
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
				testPoint(39.9042, 116.4074), // Beijing
				testPoint(39.9142, 116.4074),
				testPoint(39.9142, 116.4174),
				testPoint(39.9042, 116.4174),
			},
		},
	}
}

func TestNewPoint(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		lon     float64
		wantErr bool
	}{
		{"valid point", 39.9042, 116.4074, false},
		{"equator", 0, 0, false},
		{"north pole", 90, 0, false},
		{"south pole", -90, 0, false},
		{"international date line", 0, 180, false},
		{"invalid latitude too high", 91, 0, true},
		{"invalid latitude too low", -91, 0, true},
		{"invalid longitude too high", 0, 181, true},
		{"invalid longitude too low", 0, -181, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPoint(tt.lat, tt.lon)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPoint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFenceItem_IsActiveAt(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	fence := FenceItem{
		ID:       "test-001",
		StartTS:  past.Unix(),
		EndTS:    future.Unix(),
		Priority: 10,
	}

	if !fence.IsActiveAt(now) {
		t.Error("fence should be active at current time")
	}

	if fence.IsActiveAt(past.Add(-1 * time.Second)) {
		t.Error("fence should not be active before start time")
	}

	if fence.IsActiveAt(future.Add(1 * time.Second)) {
		t.Error("fence should not be active after end time")
	}
}

func TestFenceItem_IsActiveNow(t *testing.T) {
	now := time.Now()

	t.Run("active fence", func(t *testing.T) {
		fence := FenceItem{
			ID:       "test-001",
			StartTS:  now.Add(-1 * time.Hour).Unix(),
			EndTS:    now.Add(1 * time.Hour).Unix(),
			Priority: 10,
		}
		if !fence.IsActiveNow() {
			t.Error("fence should be active now")
		}
	})

	t.Run("expired fence", func(t *testing.T) {
		fence := FenceItem{
			ID:       "test-002",
			StartTS:  now.Add(-2 * time.Hour).Unix(),
			EndTS:    now.Add(-1 * time.Hour).Unix(),
			Priority: 10,
		}
		if fence.IsActiveNow() {
			t.Error("expired fence should not be active now")
		}
	})

	t.Run("no end time", func(t *testing.T) {
		fence := FenceItem{
			ID:       "test-003",
			StartTS:  now.Add(-1 * time.Hour).Unix(),
			EndTS:    0, // No expiry
			Priority: 10,
		}
		if !fence.IsActiveNow() {
			t.Error("fence with no end time should be active")
		}
	})
}

func TestBoundingBox_Contains(t *testing.T) {
	bbox := BoundingBox{
		MinLat: 39.0,
		MaxLat: 40.0,
		MinLon: 116.0,
		MaxLon: 117.0,
	}

	tests := []struct {
		name string
		lat  float64
		lon  float64
		want bool
	}{
		{"inside", 39.5, 116.5, true},
		{"on boundary", 39.0, 116.5, true},
		{"below", 38.9, 116.5, false},
		{"above", 40.1, 116.5, false},
		{"left", 39.5, 115.9, false},
		{"right", 39.5, 117.1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Point{Latitude: tt.lat, Longitude: tt.lon}
			if got := bbox.Contains(p); got != tt.want {
				t.Errorf("BoundingBox.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeometry_ContainsPoint_Polygon(t *testing.T) {
	t.Run("square polygon", func(t *testing.T) {
		// Square around Beijing
		poly := Geometry{
			Polygon: []Point{
				{Latitude: 39.9, Longitude: 116.4},
				{Latitude: 39.9, Longitude: 116.5},
				{Latitude: 40.0, Longitude: 116.5},
				{Latitude: 40.0, Longitude: 116.4},
			},
		}

		// Point inside
		inside := Point{Latitude: 39.95, Longitude: 116.45}
		if !poly.ContainsPoint(inside) {
			t.Error("point inside polygon should be contained")
		}

		// Point outside
		outside := Point{Latitude: 40.1, Longitude: 116.45}
		if poly.ContainsPoint(outside) {
			t.Error("point outside polygon should not be contained")
		}
	})

	t.Run("triangle polygon", func(t *testing.T) {
		poly := Geometry{
			Polygon: []Point{
				{Latitude: 0, Longitude: 0},
				{Latitude: 10, Longitude: 0},
				{Latitude: 5, Longitude: 10},
			},
		}

		inside := Point{Latitude: 5, Longitude: 3}
		if !poly.ContainsPoint(inside) {
			t.Error("point inside triangle should be contained")
		}

		outside := Point{Latitude: 10, Longitude: 10}
		if poly.ContainsPoint(outside) {
			t.Error("point outside triangle should not be contained")
		}
	})
}

func TestGeometry_ContainsPoint_Circle(t *testing.T) {
	center := Point{Latitude: 39.9042, Longitude: 116.4074}
	circle := Geometry{
		CircleCenter: &center,
		CircleRadius: 1000, // 1 km
	}

	// Point at center (should be inside)
	if !circle.ContainsPoint(center) {
		t.Error("center point should be inside circle")
	}

	// Point approximately 1km away (should be near boundary)
	// 0.01 degrees is roughly 1.1km at this latitude
	nearEdge := Point{Latitude: 39.9142, Longitude: 116.4074}
	result := circle.ContainsPoint(nearEdge)
	// This should be on or very close to the boundary
	if !result {
		// Allow some tolerance due to coordinate conversion
		t.Log("point near edge may be outside due to coordinate approximation")
	}

	// Point clearly outside
	far := Point{Latitude: 40.0, Longitude: 116.4074}
	if circle.ContainsPoint(far) {
		t.Error("far point should be outside circle")
	}
}

func TestGeometry_ContainsPoint_BBox(t *testing.T) {
	bbox := Geometry{
		BBox: &BoundingBox{
			MinLat: 39.0,
			MaxLat: 40.0,
			MinLon: 116.0,
			MaxLon: 117.0,
		},
	}

	inside := Point{Latitude: 39.5, Longitude: 116.5}
	if !bbox.ContainsPoint(inside) {
		t.Error("point inside bbox should be contained")
	}

	outside := Point{Latitude: 40.5, Longitude: 116.5}
	if bbox.ContainsPoint(outside) {
		t.Error("point outside bbox should not be contained")
	}
}

func TestHaversineDistance(t *testing.T) {
	// Beijing to Shanghai
	beijing := Point{Latitude: 39.9042, Longitude: 116.4074}
	shanghai := Point{Latitude: 31.2304, Longitude: 121.4737}

	distance := haversineDistance(beijing, shanghai)

	// Should be approximately 1,067 km
	expectedKm := 1067
	actualKm := distance / 1000

	diff := math.Abs(actualKm - float64(expectedKm))
	if diff > 50 { // Allow 50km tolerance
		t.Errorf("distance = %.0f km, want approximately %d km", actualKm, expectedKm)
	}
}

func TestFenceItem_ContainsPoint(t *testing.T) {
	fence := testTemporaryFence()

	// Point inside the Beijing polygon
	inside := Point{Latitude: 39.909, Longitude: 116.412}
	if !fence.ContainsPoint(inside) {
		t.Error("point should be inside fence")
	}

	// Point clearly outside
	outside := Point{Latitude: 0, Longitude: 0}
	if fence.ContainsPoint(outside) {
		t.Error("point should be outside fence")
	}
}

func TestCheckFences(t *testing.T) {
	now := time.Now()

	fences := []FenceItem{
		{
			ID:       "perm-no-fly",
			Type:     FenceTypePermanentNoFly,
			StartTS:  0,
			EndTS:    0,
			Priority: 100,
			Name:     "Airport",
			Geometry: Geometry{
				Polygon: []Point{
					{Latitude: 39.9, Longitude: 116.4},
					{Latitude: 39.9, Longitude: 116.5},
					{Latitude: 40.0, Longitude: 116.5},
					{Latitude: 40.0, Longitude: 116.4},
				},
			},
		},
		{
			ID:       "temp-restriction",
			Type:     FenceTypeTempRestriction,
			StartTS:  now.Add(-1 * time.Hour).Unix(),
			EndTS:    now.Add(1 * time.Hour).Unix(),
			Priority: 50,
			Name:     "Event zone",
			Geometry: Geometry{
				Polygon: []Point{
					{Latitude: 39.8, Longitude: 116.3},
					{Latitude: 39.8, Longitude: 116.4},
					{Latitude: 39.9, Longitude: 116.4},
					{Latitude: 39.9, Longitude: 116.3},
				},
			},
		},
	}

	t.Run("inside no-fly zone", func(t *testing.T) {
		p := Point{Latitude: 39.95, Longitude: 116.45}
		result := CheckFences(fences, p)

		if result.Allowed {
			t.Error("should not be allowed in no-fly zone")
		}

		if result.Restriction == nil {
			t.Error("should have a restriction")
		}

		if result.Restriction.ID != "perm-no-fly" {
			t.Errorf("restriction ID = %s, want perm-no-fly", result.Restriction.ID)
		}
	})

	t.Run("inside temp zone", func(t *testing.T) {
		p := Point{Latitude: 39.85, Longitude: 116.35}
		result := CheckFences(fences, p)

		if result.Allowed {
			t.Error("should not be allowed in temp restriction zone")
		}

		if result.Restriction.ID != "temp-restriction" {
			t.Errorf("restriction ID = %s, want temp-restriction", result.Restriction.ID)
		}
	})

	t.Run("outside all zones", func(t *testing.T) {
		p := Point{Latitude: 0, Longitude: 0}
		result := CheckFences(fences, p)

		if !result.Allowed {
			t.Error("should be allowed outside all zones")
		}

		if len(result.MatchingFences) != 0 {
			t.Errorf("matching fences = %d, want 0", len(result.MatchingFences))
		}
	})
}

func TestFenceItem_GetAltitudeLimit(t *testing.T) {
	now := time.Now()
	// Create a valid test polygon containing origin
	testPolygon := []Point{
		{Latitude: -1, Longitude: -1},
		{Latitude: -1, Longitude: 1},
		{Latitude: 1, Longitude: 1},
		{Latitude: 1, Longitude: -1},
	}

	t.Run("permanent no-fly zone", func(t *testing.T) {
		fence := FenceItem{
			Type:     FenceTypePermanentNoFly,
			StartTS:  0,
			EndTS:    0,
			Priority: 100,
			Geometry: Geometry{
				Polygon: testPolygon,
			},
		}

		limit := fence.GetAltitudeLimit(Point{})
		if limit != -1 {
			t.Errorf("limit = %d, want -1 (complete no-fly)", limit)
		}
	})

	t.Run("altitude limit zone", func(t *testing.T) {
		fence := FenceItem{
			Type:        FenceTypeAltitudeLimit,
			StartTS:     0,
			EndTS:       0,
			Priority:    30,
			MaxAltitude: 120,
			Geometry: Geometry{
				Polygon: testPolygon,
			},
		}

		limit := fence.GetAltitudeLimit(Point{})
		if limit != 120 {
			t.Errorf("limit = %d, want 120", limit)
		}
	})

	t.Run("inactive fence", func(t *testing.T) {
		fence := FenceItem{
			Type:     FenceTypeAltitudeLimit,
			StartTS:  now.Add(-2 * time.Hour).Unix(),
			EndTS:    now.Add(-1 * time.Hour).Unix(),
			Priority: 30,
			Geometry: Geometry{
				Polygon: testPolygon,
			},
		}

		limit := fence.GetAltitudeLimit(Point{})
		if limit != 0 {
			t.Errorf("inactive fence limit = %d, want 0", limit)
		}
	})
}

func TestFenceType_String(t *testing.T) {
	tests := []struct {
		fenceType FenceType
		want      string
	}{
		{FenceTypeTempRestriction, "TEMP_RESTRICTION"},
		{FenceTypePermanentNoFly, "PERMANENT_NO_FLY"},
		{FenceTypeAltitudeLimit, "ALTITUDE_LIMIT"},
		{FenceTypeAltitudeMinimum, "ALTITUDE_MINIMUM"},
		{FenceTypeSpeedLimit, "SPEED_LIMIT"},
		{FenceType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.fenceType.String(); got != tt.want {
				t.Errorf("FenceType.String() = %s, want %s", got, tt.want)
			}
		})
	}
}
