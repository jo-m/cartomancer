// Package forecast loads GRIB2 weather forecast data from the database and
// provides point sampling by variable, time, and location.
package forecast

import "errors"

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
