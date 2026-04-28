package forecast

import (
	"errors"
	"fmt"

	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// LiveStepM is the interpolation step used by the live forecast API endpoint.
// It is finer than [SummarizerStepM] because the time series is rendered
// directly to the client.
const LiveStepM = 200.0

// SummarizerStepM is the interpolation step used by the offline summarizer
// job. It trades some resolution for fewer points to keep the per-track work
// bounded when summarising the entire track corpus.
const SummarizerStepM = 500.0

// ErrPolylineMissing is returned by [InterpolatedTrackPoints] when the track
// row has no precomputed 50 m preview polyline. The column is populated on
// upload and by the backfill job, so an empty value indicates a track that
// has not been backfilled yet.
var ErrPolylineMissing = errors.New("forecast: polyline_dp50m_varint not backfilled")

// InterpolatedTrackPoints decodes the track's 50 m preview polyline and
// returns a sequence of points spaced at fixed stepM-metre intervals along
// the track. The cumulative distance is set on each returned point.
//
// Fixed-step interpolation is necessary because the Douglas-Peucker
// simplification used to produce the 50 m polyline can leave very long
// straight segments between consecutive vertices; sampling those directly
// would alias the forecast time series.
//
// Returns [ErrPolylineMissing] when the column is empty. Returns nil with no
// error when the polyline contains fewer than two points.
func InterpolatedTrackPoints(t db.Track, stepM float64) (track.Points, error) {
	if len(t.PolylineDp50mVarint) == 0 {
		return nil, ErrPolylineMissing
	}
	if stepM <= 0 {
		return nil, fmt.Errorf("forecast: stepM must be positive, got %v", stepM)
	}

	pts, err := track.DecodeVarint(t.PolylineDp50mVarint)
	if err != nil {
		return nil, fmt.Errorf("decode preview polyline: %w", err)
	}

	interps := pts.InterpolateByDistance(stepM)
	if len(interps) == 0 {
		return nil, nil
	}

	out := make(track.Points, len(interps))
	for i, ip := range interps {
		out[i] = track.Point{
			Lat:      ip.Lat,
			Lon:      ip.Lon,
			Distance: ip.DistanceM,
		}
	}
	return out, nil
}
