package forecast

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"jo-m.ch/go/cartomancer/internal/pkg/db/forecastdb"
	"jo-m.ch/go/cartomancer/internal/pkg/grib2"
)

// Handle holds forecast values sampled at specific grid indices only.
// Each GRIB2 file is loaded, sampled at the pre-registered locations, and
// discarded, so peak memory is proportional to one GRIB2 message plus the
// sparse output rather than to all messages combined.
type Handle struct {
	// values is keyed by variable name; each slice is sorted by valid time.
	// The inner float32 slice has one entry per registered location.
	values map[string][]timedValues

	// Attribution is the human-readable data source credit.
	Attribution string
	// AttributionHref is the URL for the data source.
	AttributionHref string
}

// timedValues holds sampled values for one time step at all registered locations.
type timedValues struct {
	validTime      time.Time
	validUntilTime time.Time
	vals           []float32
}

// Load queries the database for GRIB2 files matching the given time window and
// bounding box, decodes them, and returns a [Handle] for point sampling.
// Only the values at grid points nearest to the given (lat, lon) locations are
// retained; each GRIB2 file is parsed and discarded individually.
//
// Parameters:
//   - d: forecast database.
//   - start, end: the time window to load.
//   - bbox: spatial bounding box filter.
//   - lats, lons: the sample locations (must have equal length).
//
// Returns [ErrNoData] if no files match, or [ErrIncomplete] alongside a usable
// handle if the data only partially covers [start, end].
func Load(ctx context.Context, d *forecastdb.DB, start, end time.Time, bbox BBox, lats, lons []float64) (*Handle, error) {
	q := d.QueryRO()

	forecastRow, err := q.GetLatestForecast(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoData
	}
	if err != nil {
		return nil, fmt.Errorf("query forecast: %w", err)
	}

	grid, err := grib2.ParseGrid(bytes.NewReader(forecastRow.HorizontalGridFile))
	if err != nil {
		return nil, fmt.Errorf("parse grid constants: %w", err)
	}

	// Pre-compute nearest grid indices for each sample location.
	gridIndices := make([]int, len(lats))
	for i := range lats {
		gridIndices[i] = grid.NearestIndex(lats[i], lons[i])
	}

	// Fetch only metadata (no blob) to discover which files exist.
	metas, err := q.ListForecastFileIDsForWindow(ctx, forecastdb.ListForecastFileIDsForWindowParams{
		Start:  start,
		End:    end,
		MaxLat: sql.NullFloat64{Float64: bbox.MaxLat, Valid: true},
		MinLat: sql.NullFloat64{Float64: bbox.MinLat, Valid: true},
		MaxLon: sql.NullFloat64{Float64: bbox.MaxLon, Valid: true},
		MinLon: sql.NullFloat64{Float64: bbox.MinLon, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("query forecast file metadata: %w", err)
	}
	if len(metas) == 0 {
		return nil, ErrNoData
	}

	// Load each GRIB2 file individually, sample at grid indices, then discard.
	values := make(map[string][]timedValues)
	for _, meta := range metas {
		blob, blobErr := q.GetForecastFileBlob(ctx, meta.ID)
		if blobErr != nil {
			return nil, fmt.Errorf("get forecast file %d: %w", meta.ID, blobErr)
		}

		msgs, parseErr := grib2.Parse(bytes.NewReader(blob))
		if parseErr != nil {
			return nil, fmt.Errorf("parse GRIB2 for variable=%s validTime=%s: %w",
				meta.Variable, meta.ValidTime.Format(time.RFC3339), parseErr)
		}
		// blob is now eligible for GC.

		for _, m := range msgs {
			sampled := make([]float32, len(gridIndices))
			for i, idx := range gridIndices {
				if idx < 0 || idx >= len(m.Values) {
					sampled[i] = float32(math.NaN())
				} else {
					sampled[i] = m.Values[idx]
				}
			}

			values[meta.Variable] = append(values[meta.Variable], timedValues{
				validTime:      meta.ValidTime,
				validUntilTime: meta.ValidUntilTime,
				vals:           sampled,
			})
		}
		// msgs (with large Values slices) now eligible for GC.
	}

	// Sort each variable's entries by valid time.
	for v := range values {
		sort.Slice(values[v], func(i, j int) bool {
			return values[v][i].validTime.Before(values[v][j].validTime)
		})
	}

	h := &Handle{
		values:          values,
		Attribution:     forecastRow.Attribution,
		AttributionHref: forecastRow.AttributionHref,
	}

	if !h.coversWindow(start, end) {
		return h, ErrIncomplete
	}
	return h, nil
}

// coversWindow checks whether the loaded data spans [start, end] for at least one variable.
func (h *Handle) coversWindow(start, end time.Time) bool {
	for _, entries := range h.values {
		if len(entries) == 0 {
			continue
		}
		first := entries[0].validTime
		lastUntil := entries[len(entries)-1].validUntilTime
		if !first.After(start) && !lastUntil.Before(end) {
			return true
		}
	}
	return false
}

// Variables returns the list of variable names available in this handle.
func (h *Handle) Variables() []string {
	out := make([]string, 0, len(h.values))
	for v := range h.values {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// Sample returns the forecast value for the given variable at the given time
// and pre-registered location index. Returns NaN if the variable is unknown,
// no message covers the time, or the location index is out of range.
func (h *Handle) Sample(variable string, t time.Time, locationIdx int) float32 {
	entries, ok := h.values[variable]
	if !ok || len(entries) == 0 {
		return float32(math.NaN())
	}

	idx := sort.Search(len(entries), func(i int) bool {
		return entries[i].validTime.After(t)
	}) - 1

	if idx < 0 {
		return float32(math.NaN())
	}

	if !t.Before(entries[idx].validUntilTime) {
		return float32(math.NaN())
	}

	if locationIdx < 0 || locationIdx >= len(entries[idx].vals) {
		return float32(math.NaN())
	}

	return entries[idx].vals[locationIdx]
}
