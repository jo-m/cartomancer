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
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
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
// referenceTime and intervalStart are kept so the load-time post-processing
// pass can de-average variables that ICON-CH1-EPS publishes as running means
// (or running totals) from the model reference time. They are not used after
// that pass.
type timedValues struct {
	validTime      time.Time
	validUntilTime time.Time
	referenceTime  time.Time
	intervalStart  time.Time
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
				referenceTime:  m.ReferenceTime,
				intervalStart:  m.IntervalStart,
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

	// De-average / differentiate running-mean and running-accumulation
	// variables in place so downstream sampling sees per-step values.
	deaverageRunningAggregates(values)

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

// deaverageRunningAggregates converts running-mean and running-accumulation
// variables (those whose AggregationStart is the model reference time) from
// the [referenceTime, validTime] window form ICON-CH1-EPS publishes into
// per-step values.
//
// The transformation is in-place. Entries that cannot be de-aggregated with
// confidence (missing same-run predecessor, mismatched intervalStart, gap in
// reference time, et cetera) have their location values replaced with NaN.
// Instantaneous variables and variables not in the generated [vars.Variables]
// list are left untouched.
func deaverageRunningAggregates(values map[string][]timedValues) {
	for name, entries := range values {
		v, ok := vars.ByName(name)
		if !ok {
			continue
		}
		switch {
		case v.IsRunningMeanFromReferenceTime():
			deaverageRunningMean(entries)
		case v.IsRunningAccumulationFromReferenceTime():
			differentiateRunningAccumulation(entries)
		}
	}
}

// deaverageRunningMean turns each entry's running mean over
// [referenceTime, validTime] into the per-step mean over
// [prevValidTime, validTime].
//
// The iteration is descending so each de-averaging step still sees the
// still-running-mean predecessor value rather than an already-de-averaged
// one.
//
// The per-step value is
//
//	(L_curr * curr - L_prev * prev) / (L_curr - L_prev)
//
// where L_x is the length of the running window [referenceTime, validTime] in
// hours. An entry with no usable same-run predecessor whose interval is not
// already a single step is replaced with NaN.
func deaverageRunningMean(entries []timedValues) {
	for i := len(entries) - 1; i >= 0; i-- {
		curr := entries[i]
		if !curr.intervalStart.Equal(curr.referenceTime) {
			// Variables declared as "Aggregation Start: Reference Time" must
			// have intervalStart == referenceTime; treat anything else as
			// corrupt and refuse to fabricate a value.
			nanOut(entries[i].vals)
			continue
		}
		if curr.validTime.Equal(curr.referenceTime) {
			// Zero-length interval at lead 0; the value is whatever was
			// published (typically 0) and there is nothing to de-average.
			continue
		}

		lCurr := curr.validTime.Sub(curr.referenceTime).Hours()
		if i == 0 {
			// No predecessor: the entry is only a per-step value if its
			// window is already one hour wide. Otherwise we cannot recover a
			// per-step mean.
			if lCurr != 1 {
				nanOut(entries[i].vals)
			}
			continue
		}

		prev := entries[i-1]
		if !prev.referenceTime.Equal(curr.referenceTime) ||
			!prev.intervalStart.Equal(curr.intervalStart) ||
			!prev.validTime.Before(curr.validTime) {
			// Different run or mismatched window start: cannot de-average.
			nanOut(entries[i].vals)
			continue
		}

		lPrev := prev.validTime.Sub(prev.referenceTime).Hours()
		dl := lCurr - lPrev
		if dl <= 0 {
			nanOut(entries[i].vals)
			continue
		}

		for j := range entries[i].vals {
			pv := float64(prev.vals[j])
			cv := float64(curr.vals[j])
			if math.IsNaN(pv) || math.IsNaN(cv) {
				entries[i].vals[j] = float32(math.NaN())
				continue
			}
			entries[i].vals[j] = float32((lCurr*cv - lPrev*pv) / dl)
		}
	}
}

// differentiateRunningAccumulation turns each entry's running total over
// [referenceTime, validTime] into the per-step total over
// [prevValidTime, validTime] via simple subtraction.
//
// The iteration is descending so each step still sees the original
// running-total predecessor value. The result keeps the original unit but
// now denotes "amount accumulated during this step", not "amount accumulated
// since reference time".
func differentiateRunningAccumulation(entries []timedValues) {
	for i := len(entries) - 1; i >= 0; i-- {
		curr := entries[i]
		if !curr.intervalStart.Equal(curr.referenceTime) {
			nanOut(entries[i].vals)
			continue
		}
		if curr.validTime.Equal(curr.referenceTime) {
			// Zero-length interval at lead 0; accumulated amount is 0 by
			// definition. Leave whatever the file published in place.
			continue
		}

		if i == 0 {
			// Without a predecessor we cannot recover a per-step total
			// unless the running window is already one step wide. The
			// running-total form has no way of knowing the step length, so
			// keep the value as-is only when the file itself spans 1h.
			lCurr := curr.validTime.Sub(curr.referenceTime).Hours()
			if lCurr != 1 {
				nanOut(entries[i].vals)
			}
			continue
		}

		prev := entries[i-1]
		if !prev.referenceTime.Equal(curr.referenceTime) ||
			!prev.intervalStart.Equal(curr.intervalStart) ||
			!prev.validTime.Before(curr.validTime) {
			nanOut(entries[i].vals)
			continue
		}

		for j := range entries[i].vals {
			pv := float64(prev.vals[j])
			cv := float64(curr.vals[j])
			if math.IsNaN(pv) || math.IsNaN(cv) {
				entries[i].vals[j] = float32(math.NaN())
				continue
			}
			entries[i].vals[j] = float32(cv - pv)
		}
	}
}

// nanOut sets every entry in vals to NaN.
func nanOut(vals []float32) {
	for j := range vals {
		vals[j] = float32(math.NaN())
	}
}
