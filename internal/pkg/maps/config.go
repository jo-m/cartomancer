// Package maps downloads and manages PMTiles map extracts from protomaps.com.
package maps

import (
	"errors"
	"fmt"
)

// DefaultBbox is a bounding box covering approximately Switzerland and nearby border areas.
const DefaultBbox = "5.5,45.5,11.0,48.2"

// DefaultMaxZoom is the default maximum zoom level for extracted tiles.
const DefaultMaxZoom = 8

// MapsConfig contains configuration for the maps downloader.
// It has struct tags compatible with [github.com/alexflint/go-arg].
//
//revive:disable:exported Naming necessary for struct embedding.
type MapsConfig struct {
	// MapsDir is the directory where extracted PMTiles files are stored.
	// Defaults to empty, which disables the maps downloader.
	MapsDir string `arg:"--maps-dir,env:MAPS_DIR" default:"" help:"Directory for PMTiles map extracts (empty to disable)" placeholder:"DIR"`
	// MapsBbox is the bounding box for the map extract in min_lon,min_lat,max_lon,max_lat format.
	MapsBbox string `arg:"--maps-bbox,env:MAPS_BBOX" default:"5.5,45.5,11.0,48.2" help:"Bounding box for map extract (min_lon,min_lat,max_lon,max_lat)" placeholder:"BBOX"`
	// MapsMaxZoom is the maximum zoom level to extract.
	MapsMaxZoom int `arg:"--maps-maxzoom,env:MAPS_MAXZOOM" default:"8" help:"Maximum zoom level for map extract" placeholder:"Z"`
}

// Enabled returns true if the maps downloader is configured.
func (c *MapsConfig) Enabled() bool {
	return c.MapsDir != ""
}

// Validate checks for basic configuration errors.
func (c *MapsConfig) Validate() error {
	if !c.Enabled() {
		return nil
	}
	if c.MapsMaxZoom < 0 || c.MapsMaxZoom > 15 {
		return fmt.Errorf("--maps-maxzoom / MAPS_MAXZOOM must be between 0 and 15, got %d", c.MapsMaxZoom)
	}
	if c.MapsBbox == "" {
		return errors.New("--maps-bbox / MAPS_BBOX must not be empty when maps are enabled")
	}
	return nil
}
