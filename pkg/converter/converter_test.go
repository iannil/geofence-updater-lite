package converter

import (
	"testing"

	"github.com/iannil/geofence-updater-lite/pkg/geofence"
	pb "github.com/iannil/geofence-updater-lite/pkg/protocol/protobuf"
)

func TestFenceItemFromProto_Nil(t *testing.T) {
	result := FenceItemFromProto(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestFenceItemFromProto_Basic(t *testing.T) {
	pbItem := &pb.FenceItem{
		Id:                "test-fence-001",
		Type:              pb.FenceType_FENCE_TYPE_PERMANENT_NO_FLY,
		StartTs:           1000,
		EndTs:             2000,
		Priority:          50,
		MaxAltitudeMeters: 120,
		MaxSpeedMps:       10,
		Name:              "Test Fence",
		Description:       "A test fence",
		Signature:         []byte("sig"),
		KeyId:             "key123",
	}

	result := FenceItemFromProto(pbItem)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ID != pbItem.Id {
		t.Errorf("ID = %s, want %s", result.ID, pbItem.Id)
	}
	if result.Type != geofence.FenceType(pbItem.Type) {
		t.Errorf("Type = %d, want %d", result.Type, pbItem.Type)
	}
	if result.StartTS != pbItem.StartTs {
		t.Errorf("StartTS = %d, want %d", result.StartTS, pbItem.StartTs)
	}
	if result.EndTS != pbItem.EndTs {
		t.Errorf("EndTS = %d, want %d", result.EndTS, pbItem.EndTs)
	}
	if result.Priority != pbItem.Priority {
		t.Errorf("Priority = %d, want %d", result.Priority, pbItem.Priority)
	}
	if result.MaxAltitude != pbItem.MaxAltitudeMeters {
		t.Errorf("MaxAltitude = %d, want %d", result.MaxAltitude, pbItem.MaxAltitudeMeters)
	}
	if result.MaxSpeed != pbItem.MaxSpeedMps {
		t.Errorf("MaxSpeed = %d, want %d", result.MaxSpeed, pbItem.MaxSpeedMps)
	}
	if result.Name != pbItem.Name {
		t.Errorf("Name = %s, want %s", result.Name, pbItem.Name)
	}
	if result.Description != pbItem.Description {
		t.Errorf("Description = %s, want %s", result.Description, pbItem.Description)
	}
}

func TestFenceItemFromProto_Polygon(t *testing.T) {
	pbItem := &pb.FenceItem{
		Id: "polygon-fence",
		Geometry: &pb.Geometry{
			Shape: &pb.Geometry_Polygon{
				Polygon: &pb.Polygon{
					Coordinates: []*pb.Point{
						{Latitude: 39.0, Longitude: 116.0},
						{Latitude: 39.0, Longitude: 117.0},
						{Latitude: 40.0, Longitude: 117.0},
						{Latitude: 40.0, Longitude: 116.0},
					},
				},
			},
		},
	}

	result := FenceItemFromProto(pbItem)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Geometry.Polygon) != 4 {
		t.Errorf("Polygon length = %d, want 4", len(result.Geometry.Polygon))
	}
	if result.Geometry.Polygon[0].Latitude != 39.0 {
		t.Errorf("First point lat = %f, want 39.0", result.Geometry.Polygon[0].Latitude)
	}
	if result.Geometry.Polygon[0].Longitude != 116.0 {
		t.Errorf("First point lon = %f, want 116.0", result.Geometry.Polygon[0].Longitude)
	}
}

func TestFenceItemFromProto_Circle(t *testing.T) {
	pbItem := &pb.FenceItem{
		Id: "circle-fence",
		Geometry: &pb.Geometry{
			Shape: &pb.Geometry_Circle{
				Circle: &pb.Circle{
					Center:       &pb.Point{Latitude: 39.5, Longitude: 116.5},
					RadiusMeters: 1000.0,
				},
			},
		},
	}

	result := FenceItemFromProto(pbItem)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Geometry.CircleCenter == nil {
		t.Fatal("CircleCenter should not be nil")
	}
	if result.Geometry.CircleCenter.Latitude != 39.5 {
		t.Errorf("Circle center lat = %f, want 39.5", result.Geometry.CircleCenter.Latitude)
	}
	if result.Geometry.CircleRadius != 1000.0 {
		t.Errorf("CircleRadius = %f, want 1000.0", result.Geometry.CircleRadius)
	}
}

func TestFenceItemFromProto_BBox(t *testing.T) {
	pbItem := &pb.FenceItem{
		Id: "bbox-fence",
		Geometry: &pb.Geometry{
			Shape: &pb.Geometry_Bbox{
				Bbox: &pb.BoundingBox{
					MinLat: 39.0,
					MinLon: 116.0,
					MaxLat: 40.0,
					MaxLon: 117.0,
				},
			},
		},
	}

	result := FenceItemFromProto(pbItem)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Geometry.BBox == nil {
		t.Fatal("BBox should not be nil")
	}
	if result.Geometry.BBox.MinLat != 39.0 {
		t.Errorf("BBox.MinLat = %f, want 39.0", result.Geometry.BBox.MinLat)
	}
	if result.Geometry.BBox.MaxLon != 117.0 {
		t.Errorf("BBox.MaxLon = %f, want 117.0", result.Geometry.BBox.MaxLon)
	}
}

func TestFenceItemToProto_Nil(t *testing.T) {
	result := FenceItemToProto(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestFenceItemToProto_Basic(t *testing.T) {
	item := &geofence.FenceItem{
		ID:          "test-fence-002",
		Type:        geofence.FenceTypeTempRestriction,
		StartTS:     1000,
		EndTS:       2000,
		Priority:    75,
		MaxAltitude: 200,
		MaxSpeed:    15,
		Name:        "Proto Test",
		Description: "Test to proto",
		Signature:   []byte("signature"),
		KeyID:       "key456",
	}

	result := FenceItemToProto(item)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Id != item.ID {
		t.Errorf("Id = %s, want %s", result.Id, item.ID)
	}
	if result.Type != pb.FenceType(item.Type) {
		t.Errorf("Type = %d, want %d", result.Type, item.Type)
	}
	if result.Priority != item.Priority {
		t.Errorf("Priority = %d, want %d", result.Priority, item.Priority)
	}
}

func TestFenceItemToProto_Polygon(t *testing.T) {
	item := &geofence.FenceItem{
		ID: "polygon-to-proto",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 39.0, Longitude: 116.0},
				{Latitude: 40.0, Longitude: 117.0},
			},
		},
	}

	result := FenceItemToProto(item)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Geometry == nil {
		t.Fatal("Geometry should not be nil")
	}
	poly := result.Geometry.GetPolygon()
	if poly == nil {
		t.Fatal("Expected polygon geometry")
	}
	if len(poly.Coordinates) != 2 {
		t.Errorf("Polygon coordinates length = %d, want 2", len(poly.Coordinates))
	}
}

func TestFenceItemToProto_Circle(t *testing.T) {
	center := geofence.Point{Latitude: 39.5, Longitude: 116.5}
	item := &geofence.FenceItem{
		ID: "circle-to-proto",
		Geometry: geofence.Geometry{
			CircleCenter: &center,
			CircleRadius: 500.0,
		},
	}

	result := FenceItemToProto(item)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	circle := result.Geometry.GetCircle()
	if circle == nil {
		t.Fatal("Expected circle geometry")
	}
	if circle.RadiusMeters != 500.0 {
		t.Errorf("RadiusMeters = %f, want 500.0", circle.RadiusMeters)
	}
}

func TestFenceItemToProto_BBox(t *testing.T) {
	item := &geofence.FenceItem{
		ID: "bbox-to-proto",
		Geometry: geofence.Geometry{
			BBox: &geofence.BoundingBox{
				MinLat: 38.0,
				MinLon: 115.0,
				MaxLat: 41.0,
				MaxLon: 118.0,
			},
		},
	}

	result := FenceItemToProto(item)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	bbox := result.Geometry.GetBbox()
	if bbox == nil {
		t.Fatal("Expected bbox geometry")
	}
	if bbox.MinLat != 38.0 {
		t.Errorf("MinLat = %f, want 38.0", bbox.MinLat)
	}
}

func TestFenceItemRoundTrip(t *testing.T) {
	original := &geofence.FenceItem{
		ID:          "roundtrip-fence",
		Type:        geofence.FenceTypePermanentNoFly,
		StartTS:     1000,
		EndTS:       2000,
		Priority:    100,
		MaxAltitude: 150,
		MaxSpeed:    20,
		Name:        "Roundtrip Test",
		Description: "Testing roundtrip conversion",
		Signature:   []byte("test-sig"),
		KeyID:       "test-key",
		Geometry: geofence.Geometry{
			Polygon: []geofence.Point{
				{Latitude: 39.0, Longitude: 116.0},
				{Latitude: 39.0, Longitude: 117.0},
				{Latitude: 40.0, Longitude: 117.0},
				{Latitude: 40.0, Longitude: 116.0},
			},
		},
	}

	// Convert to proto and back
	pbItem := FenceItemToProto(original)
	result := FenceItemFromProto(pbItem)

	if result.ID != original.ID {
		t.Errorf("ID = %s, want %s", result.ID, original.ID)
	}
	if result.Type != original.Type {
		t.Errorf("Type = %d, want %d", result.Type, original.Type)
	}
	if result.Priority != original.Priority {
		t.Errorf("Priority = %d, want %d", result.Priority, original.Priority)
	}
	if len(result.Geometry.Polygon) != len(original.Geometry.Polygon) {
		t.Errorf("Polygon length = %d, want %d", len(result.Geometry.Polygon), len(original.Geometry.Polygon))
	}
}

func TestFenceCollectionFromProto_Nil(t *testing.T) {
	result := FenceCollectionFromProto(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestFenceCollectionFromProto_Basic(t *testing.T) {
	pbCol := &pb.FenceCollection{
		CreatedTs: 12345,
		Version:   "5",
		Items: []*pb.FenceItem{
			{Id: "fence-1"},
			{Id: "fence-2"},
		},
	}

	result := FenceCollectionFromProto(pbCol)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.CreatedTS != pbCol.CreatedTs {
		t.Errorf("CreatedTS = %d, want %d", result.CreatedTS, pbCol.CreatedTs)
	}
	if result.Version != pbCol.Version {
		t.Errorf("Version = %s, want %s", result.Version, pbCol.Version)
	}
	if len(result.Items) != 2 {
		t.Errorf("Items count = %d, want 2", len(result.Items))
	}
}

func TestFenceCollectionToProto_Nil(t *testing.T) {
	result := FenceCollectionToProto(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestFenceCollectionToProto_Basic(t *testing.T) {
	col := &geofence.FenceCollection{
		CreatedTS: 54321,
		Version:   "10",
		Items: []geofence.FenceItem{
			{ID: "item-1"},
			{ID: "item-2"},
			{ID: "item-3"},
		},
	}

	result := FenceCollectionToProto(col)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.CreatedTs != col.CreatedTS {
		t.Errorf("CreatedTs = %d, want %d", result.CreatedTs, col.CreatedTS)
	}
	if len(result.Items) != 3 {
		t.Errorf("Items count = %d, want 3", len(result.Items))
	}
}

func TestFenceDeltaFromProto_Nil(t *testing.T) {
	result := FenceDeltaFromProto(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestFenceDeltaFromProto_Basic(t *testing.T) {
	pbDelta := &pb.FenceDelta{
		RemovedIds: []string{"removed-1", "removed-2"},
		Added: []*pb.FenceItem{
			{Id: "added-1"},
		},
		Updated: []*pb.FenceItem{
			{Id: "updated-1"},
			{Id: "updated-2"},
		},
	}

	result := FenceDeltaFromProto(pbDelta)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.RemovedIDs) != 2 {
		t.Errorf("RemovedIDs count = %d, want 2", len(result.RemovedIDs))
	}
	if len(result.Added) != 1 {
		t.Errorf("Added count = %d, want 1", len(result.Added))
	}
	if len(result.Updated) != 2 {
		t.Errorf("Updated count = %d, want 2", len(result.Updated))
	}
}

func TestFenceDeltaToProto_Nil(t *testing.T) {
	result := FenceDeltaToProto(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestFenceDeltaToProto_Basic(t *testing.T) {
	delta := &geofence.FenceDelta{
		RemovedIDs: []string{"del-1"},
		Added: []geofence.FenceItem{
			{ID: "new-1"},
			{ID: "new-2"},
		},
		Updated: []geofence.FenceItem{
			{ID: "upd-1"},
		},
	}

	result := FenceDeltaToProto(delta)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.RemovedIds) != 1 {
		t.Errorf("RemovedIds count = %d, want 1", len(result.RemovedIds))
	}
	if len(result.Added) != 2 {
		t.Errorf("Added count = %d, want 2", len(result.Added))
	}
	if len(result.Updated) != 1 {
		t.Errorf("Updated count = %d, want 1", len(result.Updated))
	}
}

func TestManifestFromProto_Nil(t *testing.T) {
	result := ManifestFromProto(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestManifestFromProto_Basic(t *testing.T) {
	pbManifest := &pb.Manifest{
		Version:          42,
		Timestamp:        1234567890,
		RootHash:         []byte("roothash"),
		DeltaUrl:         "/delta.bin",
		SnapshotUrl:      "/snapshot.bin",
		DeltaSize:        1000,
		SnapshotSize:     5000,
		DeltaHash:        []byte("deltahash"),
		SnapshotHash:     []byte("snaphash"),
		MinClientVersion: 100,
		Message:          "Test manifest",
		Signature:        []byte("manifestsig"),
		KeyId:            "manifest-key",
	}

	result := ManifestFromProto(pbManifest)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Version != pbManifest.Version {
		t.Errorf("Version = %d, want %d", result.Version, pbManifest.Version)
	}
	if result.DeltaURL != pbManifest.DeltaUrl {
		t.Errorf("DeltaURL = %s, want %s", result.DeltaURL, pbManifest.DeltaUrl)
	}
	if result.Message != pbManifest.Message {
		t.Errorf("Message = %s, want %s", result.Message, pbManifest.Message)
	}
}

func TestManifestToProto_Nil(t *testing.T) {
	result := ManifestToProto(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestManifestToProto_Basic(t *testing.T) {
	manifest := &geofence.Manifest{
		Version:      100,
		Timestamp:    9876543210,
		RootHash:     []byte("root"),
		DeltaURL:     "/d.bin",
		SnapshotURL:  "/s.bin",
		DeltaSize:    2000,
		SnapshotSize: 10000,
		DeltaHash:    []byte("dh"),
		SnapshotHash: []byte("sh"),
		MinClientV:   200,
		Message:      "To proto test",
		Signature:    []byte("sig"),
		KeyID:        "key",
	}

	result := ManifestToProto(manifest)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Version != manifest.Version {
		t.Errorf("Version = %d, want %d", result.Version, manifest.Version)
	}
	if result.SnapshotUrl != manifest.SnapshotURL {
		t.Errorf("SnapshotUrl = %s, want %s", result.SnapshotUrl, manifest.SnapshotURL)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	original := &geofence.Manifest{
		Version:      50,
		Timestamp:    1111111111,
		RootHash:     []byte("original-root"),
		DeltaURL:     "/patches/delta.bin",
		SnapshotURL:  "/snapshots/snap.bin",
		DeltaSize:    500,
		SnapshotSize: 2500,
		DeltaHash:    []byte("delta-hash"),
		SnapshotHash: []byte("snap-hash"),
		MinClientV:   150,
		Message:      "Roundtrip manifest",
		Signature:    []byte("roundtrip-sig"),
		KeyID:        "roundtrip-key",
	}

	pbManifest := ManifestToProto(original)
	result := ManifestFromProto(pbManifest)

	if result.Version != original.Version {
		t.Errorf("Version = %d, want %d", result.Version, original.Version)
	}
	if result.DeltaURL != original.DeltaURL {
		t.Errorf("DeltaURL = %s, want %s", result.DeltaURL, original.DeltaURL)
	}
	if result.SnapshotURL != original.SnapshotURL {
		t.Errorf("SnapshotURL = %s, want %s", result.SnapshotURL, original.SnapshotURL)
	}
	if result.Message != original.Message {
		t.Errorf("Message = %s, want %s", result.Message, original.Message)
	}
	if result.KeyID != original.KeyID {
		t.Errorf("KeyID = %s, want %s", result.KeyID, original.KeyID)
	}
}
