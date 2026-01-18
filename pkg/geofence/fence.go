package geofence

import (
	"math"
)

// ContainsPoint checks if the fence geometry contains a point.
func (g *Geometry) ContainsPoint(p Point) bool {
	if b := g.BBox; b != nil {
		return b.Contains(p)
	}
	if len(g.Polygon) > 0 {
		return pointInPolygon(p, g.Polygon)
	}
	if g.CircleCenter != nil {
		return pointInCircle(p, *g.CircleCenter, g.CircleRadius)
	}
	return false
}

// pointInPolygon implements the ray-casting algorithm to check if a point
// is inside a polygon.
func pointInPolygon(p Point, polygon []Point) bool {
	if len(polygon) < 3 {
		return false
	}

	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		vi := polygon[i]
		vj := polygon[j]

		if ((vi.Longitude > p.Longitude) != (vj.Longitude > p.Longitude)) &&
			p.Latitude < (vj.Latitude-vi.Latitude)*(p.Longitude-vi.Longitude)/(vj.Longitude-vi.Longitude+1e-10)+vi.Latitude {
			inside = !inside
		}
		j = i
	}

	return inside
}

// pointInCircle checks if a point is within a circle using Haversine distance.
func pointInCircle(p, center Point, radiusMeters float64) bool {
	distance := haversineDistance(p, center)
	return distance <= radiusMeters
}

// haversineDistance calculates the great-circle distance between two points
// on Earth using the Haversine formula.
func haversineDistance(p1, p2 Point) float64 {
	const earthRadius = 6371000.0 // Earth's radius in meters

	lat1 := degToRad(p1.Latitude)
	lon1 := degToRad(p1.Longitude)
	lat2 := degToRad(p2.Latitude)
	lon2 := degToRad(p2.Longitude)

	dLat := lat2 - lat1
	dLon := lon2 - lon1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// RestrictionLevel returns the restriction severity for a fence at a location.
// Returns 0 if no restriction, higher values indicate more severe restrictions.
func (f *FenceItem) RestrictionLevel(at Point) int32 {
	if !f.ContainsPoint(at) {
		return 0
	}
	if !f.IsActiveNow() {
		return 0
	}

	// Different fence types have different severity levels
	switch f.Type {
	case FenceTypePermanentNoFly:
		return 100
	case FenceTypeTempRestriction:
		return 80
	case FenceTypeAltitudeLimit:
		return 50
	case FenceTypeAltitudeMinimum:
		return 40
	case FenceTypeSpeedLimit:
		return 20
	default:
		return 10
	}
}

// ContainsPoint checks if the fence item's geometry contains a point.
func (f *FenceItem) ContainsPoint(p Point) bool {
	return f.Geometry.ContainsPoint(p)
}

// GetAltitudeLimit returns the maximum allowed altitude at a point.
// Returns 0 if no limit, -1 if completely forbidden.
func (f *FenceItem) GetAltitudeLimit(at Point) int32 {
	if !f.ContainsPoint(at) {
		return 0
	}
	if !f.IsActiveNow() {
		return 0
	}

	if f.Type == FenceTypePermanentNoFly || f.Type == FenceTypeTempRestriction {
		return -1 // Complete no-fly
	}

	if f.Type == FenceTypeAltitudeLimit && f.MaxAltitude > 0 {
		return int32(f.MaxAltitude)
	}

	return 0
}

// GetSpeedLimit returns the maximum allowed speed at a point in m/s.
// Returns 0 if no limit.
func (f *FenceItem) GetSpeedLimit(at Point) int32 {
	if !f.ContainsPoint(at) {
		return 0
	}
	if !f.IsActiveNow() {
		return 0
	}

	if f.Type == FenceTypeSpeedLimit && f.MaxSpeed > 0 {
		return int32(f.MaxSpeed)
	}

	return 0
}

// CheckFences checks multiple fences and returns the most restrictive result.
func CheckFences(fences []FenceItem, p Point) CheckResult {
	var highestPriority *FenceItem
	var matchingFences []FenceItem

	for i := range fences {
		f := &fences[i]
		if f.ContainsPoint(p) && f.IsActiveNow() {
			matchingFences = append(matchingFences, *f)

			if highestPriority == nil || f.Priority > highestPriority.Priority {
				highestPriority = f
			}
		}
	}

	result := CheckResult{
		Allowed:       true,
		MatchingFences: matchingFences,
	}

	if highestPriority != nil {
		if highestPriority.Type == FenceTypePermanentNoFly ||
			highestPriority.Type == FenceTypeTempRestriction {
			result.Allowed = false
			result.Restriction = highestPriority
		}
	}

	return result
}
