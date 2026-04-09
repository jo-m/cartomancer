// Package forecast loads GRIB2 weather forecast data from the database and
// provides point sampling by variable, time, and location.
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

// Sentinel errors returned by [Load].
var (
	// ErrNoData indicates that no forecast data is available for the requested constraints.
	ErrNoData = errors.New("forecast: no data available for the requested constraints")
	// ErrIncomplete indicates that forecast data only partially covers the
	// requested time window. The returned [Handle] still contains partial data.
	ErrIncomplete = errors.New("forecast: data only partially covers the requested time window")
)

// BBox defines a lat/lon bounding box in WGS84 degrees.
type BBox struct {
	MinLat, MaxLat, MinLon, MaxLon float64
}

// timedMessage pairs a parsed GRIB2 message with its validity interval for sorting.
type timedMessage struct {
	validTime      time.Time
	validUntilTime time.Time
	msg            *grib2.Message
}

// Handle holds decoded GRIB2 messages in memory, ready for point sampling.
type Handle struct {
	grid *grib2.Grid
	// messages is keyed by variable name; each slice is sorted by valid time.
	messages map[string][]timedMessage

	// Attribution is the human-readable data source credit.
	Attribution string
	// AttributionHref is the URL for the data source.
	AttributionHref string
}

// Load queries the database for GRIB2 files matching the given time window
// and bounding box, decodes them, and returns a [Handle] for point sampling.
//
// It returns [ErrNoData] if no files match at all. If the loaded files do not
// fully span [start, end], it returns [ErrIncomplete] alongside a usable Handle
// so callers can work with partial data.
func Load(ctx context.Context, d *forecastdb.DB, start, end time.Time, bbox BBox) (*Handle, error) {
	q := d.QueryRO()

	// Load the latest forecast row (contains grid constants).
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

	// Load forecast files for the time/space window.
	rows, err := q.ListForecastFilesForWindow(ctx, forecastdb.ListForecastFilesForWindowParams{
		Start:  start,
		End:    end,
		MaxLat: sql.NullFloat64{Float64: bbox.MaxLat, Valid: true},
		MinLat: sql.NullFloat64{Float64: bbox.MinLat, Valid: true},
		MaxLon: sql.NullFloat64{Float64: bbox.MaxLon, Valid: true},
		MinLon: sql.NullFloat64{Float64: bbox.MinLon, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("query forecast files: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrNoData
	}

	// Parse each GRIB2 file and group messages by variable.
	messages := make(map[string][]timedMessage)
	for _, row := range rows {
		msgs, parseErr := grib2.Parse(bytes.NewReader(row.File))
		if parseErr != nil {
			return nil, fmt.Errorf("parse GRIB2 for variable=%s validTime=%s: %w",
				row.Variable, row.ValidTime.Format(time.RFC3339), parseErr)
		}

		for _, m := range msgs {
			messages[row.Variable] = append(messages[row.Variable], timedMessage{
				validTime:      row.ValidTime,
				validUntilTime: row.ValidUntilTime,
				msg:            m,
			})
		}
	}

	// Sort each variable's messages by valid time.
	for v := range messages {
		sort.Slice(messages[v], func(i, j int) bool {
			return messages[v][i].validTime.Before(messages[v][j].validTime)
		})
	}

	h := &Handle{
		grid:            grid,
		messages:        messages,
		Attribution:     forecastRow.Attribution,
		AttributionHref: forecastRow.AttributionHref,
	}

	// Check coverage: whether loaded data spans the full requested window.
	if !h.coversWindow(start, end) {
		return h, ErrIncomplete
	}

	return h, nil
}

// coversWindow checks whether the loaded messages span the full [start, end]
// window for at least one variable. Each message covers [validTime, validUntilTime),
// so coverage requires the first message to start at or before start and the last
// message's validity to extend to or past end.
func (h *Handle) coversWindow(start, end time.Time) bool {
	for _, msgs := range h.messages {
		if len(msgs) == 0 {
			continue
		}
		first := msgs[0].validTime
		lastUntil := msgs[len(msgs)-1].validUntilTime
		if !first.After(start) && !lastUntil.Before(end) {
			return true
		}
	}
	return false
}

// Variables returns the list of variable names available in this handle.
func (h *Handle) Variables() []string {
	out := make([]string, 0, len(h.messages))
	for v := range h.messages {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// Sample returns the forecast value for the given variable at the given time
// and location. It picks the message whose validity interval [validTime, validUntilTime)
// contains t. Returns NaN if the variable is unknown, no message covers the time,
// or the point is outside the grid domain.
func (h *Handle) Sample(variable string, t time.Time, lat, lon float64) float32 {
	msgs, ok := h.messages[variable]
	if !ok || len(msgs) == 0 {
		return float32(math.NaN())
	}

	// Binary search for the rightmost message with validTime <= t.
	idx := sort.Search(len(msgs), func(i int) bool {
		return msgs[i].validTime.After(t)
	}) - 1

	if idx < 0 {
		return float32(math.NaN())
	}

	// Check that t falls within the message's validity interval.
	if !t.Before(msgs[idx].validUntilTime) {
		return float32(math.NaN())
	}

	return h.grid.ValueAt(msgs[idx].msg, lat, lon)
}
