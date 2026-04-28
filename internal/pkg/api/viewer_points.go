package api

import (
	"bytes"
	"context"
	"fmt"

	"jo-m.ch/go/cartomancer/internal/pkg/blob"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/load"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// loadViewerPoints returns a viewer-resolution point set for t.
//
// When the precomputed varint polyline column for kind is non-empty it is
// decoded and thinned by [track.Points.Subsample] to minDistM, avoiding a
// blob fetch and re-parse. Otherwise the raw blob is loaded, parsed, and
// passed through [track.Points.SimplifyForView] using epsilonM and minDistM.
//
// kind selects which precomputed polyline column to read; minDistM and
// epsilonM are the thinning parameters as documented on
// [track.PointsViewerEpsilonM] and [track.ForecastViewerEpsilonM].
func loadViewerPoints(ctx context.Context, q *db.Queries, t db.Track, kind db.PreviewPolylineKind, epsilonM, minDistM float64) (track.Points, error) {
	var encoded []byte
	switch kind {
	case db.PreviewPolyline5M:
		encoded = t.PolylineDp5mVarint
	case db.PreviewPolyline50M:
		encoded = t.PolylineDp50mVarint
	default:
		return nil, fmt.Errorf("unknown preview polyline kind: %v", kind)
	}

	if len(encoded) > 0 {
		pts, err := track.DecodeVarint(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode preview polyline: %w", err)
		}
		return pts.Subsample(minDistM), nil
	}

	b, err := blob.Get(ctx, q, t.BlobID)
	if err != nil {
		return nil, fmt.Errorf("get blob: %w", err)
	}
	src, err := load.Blob(t.OriginalFilename, bytes.NewReader(b.Content))
	if err != nil {
		return nil, fmt.Errorf("parse blob: %w", err)
	}
	tr, err := track.New(src)
	if err != nil {
		return nil, fmt.Errorf("new track: %w", err)
	}
	return tr.Points().SimplifyForView(epsilonM, minDistM), nil
}
