// Package converter provides conversion between Go types and Protobuf messages.
package converter

import (
	"github.com/iannil/geofence-updater-lite/pkg/geofence"
	pb "github.com/iannil/geofence-updater-lite/pkg/protocol/protobuf"
)

// FenceItemFromProto converts a Protobuf FenceItem to a Go FenceItem.
func FenceItemFromProto(pbItem *pb.FenceItem) *geofence.FenceItem {
	if pbItem == nil {
		return nil
	}

	item := &geofence.FenceItem{
		ID:          pbItem.Id,
		Type:        geofence.FenceType(pbItem.Type),
		StartTS:     pbItem.StartTs,
		EndTS:       pbItem.EndTs,
		Priority:    pbItem.Priority,
		MaxAltitude: pbItem.MaxAltitudeMeters,
		MaxSpeed:    pbItem.MaxSpeedMps,
		Name:        pbItem.Name,
		Description: pbItem.Description,
		Signature:   pbItem.Signature,
		KeyID:       pbItem.KeyId,
	}

	// Convert geometry
	if pbGeom := pbItem.Geometry; pbGeom != nil {
		item.Geometry = geofence.Geometry{}

		if poly := pbGeom.GetPolygon(); poly != nil {
			for _, coord := range poly.Coordinates {
				item.Geometry.Polygon = append(item.Geometry.Polygon, geofence.Point{
					Latitude:  coord.Latitude,
					Longitude: coord.Longitude,
				})
			}
		}

		if circle := pbGeom.GetCircle(); circle != nil && circle.Center != nil {
			p := geofence.Point{
				Latitude:  circle.Center.Latitude,
				Longitude: circle.Center.Longitude,
			}
			item.Geometry.CircleCenter = &p
			item.Geometry.CircleRadius = circle.RadiusMeters
		}

		if bbox := pbGeom.GetBbox(); bbox != nil {
			item.Geometry.BBox = &geofence.BoundingBox{
				MinLat: bbox.MinLat,
				MinLon: bbox.MinLon,
				MaxLat: bbox.MaxLat,
				MaxLon: bbox.MaxLon,
			}
		}
	}

	return item
}

// FenceItemToProto converts a Go FenceItem to a Protobuf FenceItem.
func FenceItemToProto(item *geofence.FenceItem) *pb.FenceItem {
	if item == nil {
		return nil
	}

	pbItem := &pb.FenceItem{
		Id:              item.ID,
		Type:            pb.FenceType(item.Type),
		StartTs:         item.StartTS,
		EndTs:           item.EndTS,
		Priority:        item.Priority,
		MaxAltitudeMeters: item.MaxAltitude,
		MaxSpeedMps:     item.MaxSpeed,
		Name:            item.Name,
		Description:     item.Description,
		Signature:       item.Signature,
		KeyId:           item.KeyID,
	}

	// Convert geometry - create the Geometry message with appropriate shape
	if len(item.Geometry.Polygon) > 0 {
		coords := make([]*pb.Point, len(item.Geometry.Polygon))
		for i, p := range item.Geometry.Polygon {
			coords[i] = &pb.Point{
				Latitude:  p.Latitude,
				Longitude: p.Longitude,
			}
		}
		pbItem.Geometry = &pb.Geometry{
			Shape: &pb.Geometry_Polygon{
				Polygon: &pb.Polygon{Coordinates: coords},
			},
		}
	} else if item.Geometry.CircleCenter != nil {
		pbItem.Geometry = &pb.Geometry{
			Shape: &pb.Geometry_Circle{
				Circle: &pb.Circle{
					Center: &pb.Point{
						Latitude:  item.Geometry.CircleCenter.Latitude,
						Longitude: item.Geometry.CircleCenter.Longitude,
					},
					RadiusMeters: item.Geometry.CircleRadius,
				},
			},
		}
	} else if item.Geometry.BBox != nil {
		pbItem.Geometry = &pb.Geometry{
			Shape: &pb.Geometry_Bbox{
				Bbox: &pb.BoundingBox{
					MinLat: item.Geometry.BBox.MinLat,
					MinLon: item.Geometry.BBox.MinLon,
					MaxLat: item.Geometry.BBox.MaxLat,
					MaxLon: item.Geometry.BBox.MaxLon,
				},
			},
		}
	}

	return pbItem
}

// FenceCollectionFromProto converts a Protobuf FenceCollection to Go.
func FenceCollectionFromProto(pbCol *pb.FenceCollection) *geofence.FenceCollection {
	if pbCol == nil {
		return nil
	}

	col := &geofence.FenceCollection{
		CreatedTS: pbCol.CreatedTs,
		Version:   pbCol.Version,
	}

	for _, pbItem := range pbCol.Items {
		if item := FenceItemFromProto(pbItem); item != nil {
			col.Items = append(col.Items, *item)
		}
	}

	return col
}

// FenceCollectionToProto converts a Go FenceCollection to Protobuf.
func FenceCollectionToProto(col *geofence.FenceCollection) *pb.FenceCollection {
	if col == nil {
		return nil
	}

	pbCol := &pb.FenceCollection{
		CreatedTs: col.CreatedTS,
		Version:   col.Version,
		Items:     make([]*pb.FenceItem, len(col.Items)),
	}

	for i, item := range col.Items {
		pbCol.Items[i] = FenceItemToProto(&item)
	}

	return pbCol
}

// FenceDeltaFromProto converts a Protobuf FenceDelta to Go.
func FenceDeltaFromProto(pbDelta *pb.FenceDelta) *geofence.FenceDelta {
	if pbDelta == nil {
		return nil
	}

	delta := &geofence.FenceDelta{
		RemovedIDs: pbDelta.RemovedIds,
	}

	for _, pbItem := range pbDelta.Added {
		if item := FenceItemFromProto(pbItem); item != nil {
			delta.Added = append(delta.Added, *item)
		}
	}

	for _, pbItem := range pbDelta.Updated {
		if item := FenceItemFromProto(pbItem); item != nil {
			delta.Updated = append(delta.Updated, *item)
		}
	}

	return delta
}

// FenceDeltaToProto converts a Go FenceDelta to Protobuf.
func FenceDeltaToProto(delta *geofence.FenceDelta) *pb.FenceDelta {
	if delta == nil {
		return nil
	}

	pbDelta := &pb.FenceDelta{
		RemovedIds: delta.RemovedIDs,
		Added:      make([]*pb.FenceItem, len(delta.Added)),
		Updated:    make([]*pb.FenceItem, len(delta.Updated)),
	}

	for i, item := range delta.Added {
		pbDelta.Added[i] = FenceItemToProto(&item)
	}

	for i, item := range delta.Updated {
		pbDelta.Updated[i] = FenceItemToProto(&item)
	}

	return pbDelta
}

// ManifestFromProto converts a Protobuf Manifest to Go.
func ManifestFromProto(pbManifest *pb.Manifest) *geofence.Manifest {
	if pbManifest == nil {
		return nil
	}

	return &geofence.Manifest{
		Version:      pbManifest.Version,
		Timestamp:    pbManifest.Timestamp,
		RootHash:     pbManifest.RootHash,
		DeltaURL:     pbManifest.DeltaUrl,
		SnapshotURL:  pbManifest.SnapshotUrl,
		DeltaSize:    pbManifest.DeltaSize,
		SnapshotSize: pbManifest.SnapshotSize,
		DeltaHash:    pbManifest.DeltaHash,
		SnapshotHash: pbManifest.SnapshotHash,
		MinClientV:   pbManifest.MinClientVersion,
		Message:      pbManifest.Message,
		Signature:    pbManifest.Signature,
		KeyID:        pbManifest.KeyId,
	}
}

// ManifestToProto converts a Go Manifest to Protobuf.
func ManifestToProto(manifest *geofence.Manifest) *pb.Manifest {
	if manifest == nil {
		return nil
	}

	return &pb.Manifest{
		Version:          manifest.Version,
		Timestamp:        manifest.Timestamp,
		RootHash:         manifest.RootHash,
		DeltaUrl:         manifest.DeltaURL,
		SnapshotUrl:      manifest.SnapshotURL,
		DeltaSize:        manifest.DeltaSize,
		SnapshotSize:     manifest.SnapshotSize,
		DeltaHash:        manifest.DeltaHash,
		SnapshotHash:     manifest.SnapshotHash,
		MinClientVersion: manifest.MinClientV,
		Message:          manifest.Message,
		Signature:        manifest.Signature,
		KeyId:            manifest.KeyID,
	}
}
