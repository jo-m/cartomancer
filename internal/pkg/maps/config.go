// Package maps downloads and manages PMTiles map extracts from protomaps.com.
package maps

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// Bbox represents a geographic bounding box with four coordinates.
type Bbox struct {
	MinLon, MinLat, MaxLon, MaxLat float64
}

// String formats the bbox as a comma-separated string for pmtiles.Extract.
func (b Bbox) String() string {
	return fmt.Sprintf("%g,%g,%g,%g", b.MinLon, b.MinLat, b.MaxLon, b.MaxLat)
}

// nullFloat64 returns a valid sql.NullFloat64 with the given value.
func nullFloat64(v float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: v, Valid: true}
}

// NullMinLon returns the MinLon as a sql.NullFloat64.
func (b Bbox) NullMinLon() sql.NullFloat64 { return nullFloat64(b.MinLon) }

// NullMinLat returns the MinLat as a sql.NullFloat64.
func (b Bbox) NullMinLat() sql.NullFloat64 { return nullFloat64(b.MinLat) }

// NullMaxLon returns the MaxLon as a sql.NullFloat64.
func (b Bbox) NullMaxLon() sql.NullFloat64 { return nullFloat64(b.MaxLon) }

// NullMaxLat returns the MaxLat as a sql.NullFloat64.
func (b Bbox) NullMaxLat() sql.NullFloat64 { return nullFloat64(b.MaxLat) }

// ParseBbox parses a "min_lon,min_lat,max_lon,max_lat" string into a [Bbox].
func ParseBbox(s string) (Bbox, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return Bbox{}, fmt.Errorf("bbox must have 4 comma-separated values, got %d", len(parts))
	}

	vals := [4]float64{}
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return Bbox{}, fmt.Errorf("bbox value %d (%q): %w", i, p, err)
		}
		vals[i] = v
	}

	return Bbox{MinLon: vals[0], MinLat: vals[1], MaxLon: vals[2], MaxLat: vals[3]}, nil
}

// MapsConfig contains configuration for the maps downloader.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type MapsConfig struct {
	// MapsBbox is the bounding box for the map extract in min_lon,min_lat,max_lon,max_lat format.
	// Empty means the entire world.
	MapsBbox string `arg:"--maps-bbox,env:MAPS_BBOX" default:"5.5,45.5,11.0,48.2" help:"Bounding box for map extract (min_lon,min_lat,max_lon,max_lat; empty for entire world)" placeholder:"BBOX"`
	// MapsMaxZoom is the maximum zoom level to extract.
	MapsMaxZoom int `arg:"--maps-maxzoom,env:MAPS_MAXZOOM" default:"10" help:"Maximum zoom level for map extract" placeholder:"Z"`
}

// ParsedBbox returns the parsed bounding box, or nil if empty (entire world).
func (c *MapsConfig) ParsedBbox() (*Bbox, error) {
	if c.MapsBbox == "" {
		return nil, nil
	}
	b, err := ParseBbox(c.MapsBbox)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// Validate checks for basic configuration errors.
func (c *MapsConfig) Validate() error {
	if c.MapsMaxZoom < 0 || c.MapsMaxZoom > 22 {
		return fmt.Errorf("--maps-maxzoom / MAPS_MAXZOOM must be between 0 and 22, got %d", c.MapsMaxZoom)
	}
	if c.MapsBbox != "" {
		if _, err := ParseBbox(c.MapsBbox); err != nil {
			return fmt.Errorf("--maps-bbox / MAPS_BBOX: %w", err)
		}
	}
	return nil
}
