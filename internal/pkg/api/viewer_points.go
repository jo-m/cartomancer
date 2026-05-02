package api

import (
	"fmt"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// loadViewerPoints returns a viewer-resolution point set for t.
//
// Decodes the precomputed varint polyline column for kind. The columns are
// NOT NULL in the schema, so an empty value here would be a programmer
// error rather than a missing-backfill situation.
func loadViewerPoints(t db.Track, kind db.PreviewPolylineKind) (track.Points, error) {
	var encoded []byte
	switch kind {
	case db.PreviewPolyline5M:
		encoded = t.PolylineDp5mVarint
	case db.PreviewPolyline50M:
		encoded = t.PolylineDp50mVarint
	default:
		return nil, fmt.Errorf("unknown preview polyline kind: %v", kind)
	}

	pts, err := track.DecodeVarint(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode preview polyline: %w", err)
	}
	return pts, nil
}
