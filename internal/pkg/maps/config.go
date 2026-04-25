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

// MapSpec describes a single map extract: an optional bounding box and a maximum zoom level.
type MapSpec struct {
	// Bbox is the geographic bounding box; nil means the entire world.
	Bbox    *Bbox
	MaxZoom int
}

// parseMapSpec parses a single spec entry in "bbox@maxzoom" format.
// An empty bbox (i.e. "@maxzoom") means the entire world.
func parseMapSpec(s string) (MapSpec, error) {
	s = strings.TrimSpace(s)
	at := strings.LastIndex(s, "@")
	if at < 0 {
		return MapSpec{}, fmt.Errorf("spec %q: missing '@' separator between bbox and zoom", s)
	}

	bboxStr := s[:at]
	zoomStr := s[at+1:]

	zoom, err := strconv.Atoi(strings.TrimSpace(zoomStr))
	if err != nil {
		return MapSpec{}, fmt.Errorf("spec %q: invalid zoom %q: %w", s, zoomStr, err)
	}
	if zoom < 0 || zoom > 22 {
		return MapSpec{}, fmt.Errorf("spec %q: zoom must be between 0 and 22, got %d", s, zoom)
	}

	var bbox *Bbox
	if strings.TrimSpace(bboxStr) != "" {
		b, err := ParseBbox(bboxStr)
		if err != nil {
			return MapSpec{}, fmt.Errorf("spec %q: %w", s, err)
		}
		bbox = &b
	}

	return MapSpec{Bbox: bbox, MaxZoom: zoom}, nil
}

// MapsConfig contains configuration for the maps downloader.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type MapsConfig struct {
	// MapsSpecs is a semicolon-separated list of map extract specifications.
	// Each entry is "bbox@maxzoom" where bbox is "min_lon,min_lat,max_lon,max_lat".
	// An empty bbox (i.e. "@maxzoom") downloads the entire world at that zoom level.
	// Example: "@7;5.5,45.5,11.0,48.2@10" downloads a world overview at zoom 7
	// and a regional extract for Switzerland at zoom 10.
	MapsSpecs string `arg:"--maps-specs,env:MAPS_SPECS" default:"@7;-16.9,27.8,-13.4,29.2@10" help:"Semicolon-separated list of map extract specs (bbox@maxzoom); empty bbox means entire world" placeholder:"SPECS"`
}

// ParsedSpecs parses MapsSpecs into a slice of [MapSpec].
func (c *MapsConfig) ParsedSpecs() ([]MapSpec, error) {
	if strings.TrimSpace(c.MapsSpecs) == "" {
		return nil, nil
	}

	entries := strings.Split(c.MapsSpecs, ";")
	specs := make([]MapSpec, 0, len(entries))
	for _, entry := range entries {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		spec, err := parseMapSpec(entry)
		if err != nil {
			return nil, fmt.Errorf("--maps-specs / MAPS_SPECS: %w", err)
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// Validate checks for basic configuration errors.
func (c *MapsConfig) Validate() error {
	if _, err := c.ParsedSpecs(); err != nil {
		return err
	}
	return nil
}
