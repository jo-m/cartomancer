package api

import (
	"context"
	"errors"
	"fmt"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// errPreviewPolylineMissing is returned by [loadViewerPoints] when the track
// row has no precomputed varint polyline for the requested kind. Callers map
// this to a 500 response; the column is populated on upload and by the
// backfill migration, so an empty value here indicates an inconsistent DB.
var errPreviewPolylineMissing = errors.New("preview polyline not backfilled")

// loadViewerPoints returns a viewer-resolution point set for t.
//
// It decodes the precomputed varint polyline column for kind and thins it
// with [track.Points.Subsample] to minDistM. epsilonM is accepted for API
// symmetry with earlier callers but is unused now that simplification is
// always done at write time. Returns [errPreviewPolylineMissing] when the
// column is empty.
func loadViewerPoints(_ context.Context, _ *db.Queries, t db.Track, kind db.PreviewPolylineKind, _, minDistM float64) (track.Points, error) {
	var encoded []byte
	switch kind {
	case db.PreviewPolyline5M:
		encoded = t.PolylineDp5mVarint
	case db.PreviewPolyline50M:
		encoded = t.PolylineDp50mVarint
	default:
		return nil, fmt.Errorf("unknown preview polyline kind: %v", kind)
	}

	if len(encoded) == 0 {
		return nil, errPreviewPolylineMissing
	}

	pts, err := track.DecodeVarint(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode preview polyline: %w", err)
	}
	return pts.Subsample(minDistM), nil
}
