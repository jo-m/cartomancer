// Package track deals with tracks.
package track

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/uber/h3-go/v4"
)

type Point struct {
	Time      time.Time
	Lat, Lon  float64
	Elevation float64
}

// MetersTo computes the great-circle distance in meters to another point.
// Elevation is ignored because it is unreliable and negligible for our purposes (ca. 0.5% at 10% grade).
func (p *Point) MetersTo(other *Point) float64 {
	return h3.GreatCircleDistanceM(
		h3.LatLng{Lat: p.Lat, Lng: p.Lon},
		h3.LatLng{Lat: other.Lat, Lng: other.Lon},
	)
}

// Sub returns p - other (lat, lon, elevation).
func (p *Point) Sub(other *Point) Point {
	return Point{
		Lat:       p.Lat - other.Lat,
		Lon:       p.Lon - other.Lon,
		Elevation: p.Elevation - other.Elevation,
	}
}

// Add returns p + other (lat, lon, elevation).
func (p *Point) Add(other *Point) Point {
	return Point{
		Lat:       p.Lat + other.Lat,
		Lon:       p.Lon + other.Lon,
		Elevation: p.Elevation + other.Elevation,
	}
}

// Mul scales lat, lon, and elevation by x.
func (p *Point) Mul(x float64) Point {
	return Point{
		Lat:       p.Lat * x,
		Lon:       p.Lon * x,
		Elevation: p.Elevation * x,
	}
}

// Interpolate returns the point at fraction x (0-1) between p and other.
func (p *Point) Interpolate(other *Point, x float64) Point {
	if x > 1 {
		panic("x cannot be > 1")
	}
	if x < 0 {
		panic("x cannot be < 0")
	}

	d := other.Sub(p)
	dx := d.Mul(x)
	return p.Add(&dx)
}

// Cell returns the H3 cell containing p at the given resolution.
func (p *Point) Cell(resolution int) h3.Cell {
	latLng := h3.LatLng{
		Lat: p.Lat,
		Lng: p.Lon,
	}
	cell, err := h3.LatLngToCell(latLng, resolution)
	if err != nil {
		panic(err)
	}
	return cell
}

type Points []Point

// PreviewOptions configures SVG rendering parameters for track previews.
type PreviewOptions struct {
	// Size is the square canvas size in pixels.
	Size int
	// StrokeWidth is the polyline stroke width in SVG user units.
	StrokeWidth float64
	// Color is the stroke color: either a CSS hex value (e.g., "#000000", "#f00")
	// or "currentColor". Invalid values are silently replaced with "currentColor".
	Color string
}

// DefaultPreviewOptions returns sensible defaults for preview rendering.
func DefaultPreviewOptions() PreviewOptions {
	return PreviewOptions{
		Size:        512,
		StrokeWidth: 1.5,
		Color:       "currentColor",
	}
}

// IsValidColor reports whether s is a valid stroke color value: either
// "currentColor" or a CSS hex color (#RGB, #RGBA, #RRGGBB, #RRGGBBAA, case-insensitive).
func IsValidColor(s string) bool {
	if s == "currentColor" {
		return true
	}
	if len(s) == 0 || s[0] != '#' {
		return false
	}
	hex := s[1:]
	n := len(hex)
	if n != 3 && n != 4 && n != 6 && n != 8 {
		return false
	}
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// Bounds represents the geographic extents of a track.
type Bounds struct {
	MinLat, MinLon float64
	MaxLat, MaxLon float64
}

// PreviewSVG renders the track as a square SVG preview image.
// opts controls the canvas size, stroke width, and color.
// If bounds is non-nil, its extents are used directly; otherwise extents are computed from pts.
// Points are subsampled so that each line segment spans approximately 5px.
func (pts Points) PreviewSVG(opts PreviewOptions, bounds *Bounds) string {
	size := opts.Size
	if len(pts) < 2 {
		return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"/>`, size, size)
	}

	var minLat, maxLat, minLon, maxLon float64
	if bounds != nil {
		minLat, maxLat = bounds.MinLat, bounds.MaxLat
		minLon, maxLon = bounds.MinLon, bounds.MaxLon
	} else {
		// Compute lat/lon extents from points.
		minLat, maxLat = pts[0].Lat, pts[0].Lat
		minLon, maxLon = pts[0].Lon, pts[0].Lon
		for _, p := range pts[1:] {
			minLat = min(minLat, p.Lat)
			maxLat = max(maxLat, p.Lat)
			minLon = min(minLon, p.Lon)
			maxLon = max(maxLon, p.Lon)
		}
	}

	dLat := maxLat - minLat
	dLon := maxLon - minLon
	if dLat == 0 && dLon == 0 {
		return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"/>`, size, size)
	}

	// Use a uniform scale so the track fits inside the square with some padding.
	pad := float64(size) * 0.05
	inner := float64(size) - 2*pad
	scale := inner / max(dLat, dLon)

	// Convert a point to SVG coordinates. Latitude is flipped (SVG y goes down).
	toX := func(p *Point) float64 {
		return pad + (p.Lon-minLon)*scale + (inner-(dLon*scale))/2
	}
	toY := func(p *Point) float64 {
		return pad + (maxLat-p.Lat)*scale + (inner-(dLat*scale))/2
	}

	// Compute total pixel-space path length to determine subsampling stride.
	totalPx := 0.0
	for i := 1; i < len(pts); i++ {
		dx := toX(&pts[i]) - toX(&pts[i-1])
		dy := toY(&pts[i]) - toY(&pts[i-1])
		totalPx += math.Sqrt(dx*dx + dy*dy)
	}

	// Target ~5px per segment.
	nSegments := max(1, int(math.Round(totalPx/5)))
	stride := max(1, len(pts)/nSegments)

	// Build polyline points string.
	var b strings.Builder
	for i := 0; i < len(pts); i += stride {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%.1f,%.1f", toX(&pts[i]), toY(&pts[i]))
	}
	// Always include the last point.
	last := &pts[len(pts)-1]
	if (len(pts)-1)%stride != 0 {
		fmt.Fprintf(&b, " %.1f,%.1f", toX(last), toY(last))
	}

	color := opts.Color
	if !IsValidColor(color) {
		color = "currentColor"
	}

	return fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`+
			`<polyline points="%s" fill="none" stroke="%s" stroke-width="%.4g" stroke-linejoin="round" stroke-linecap="round"/>`+
			`</svg>`,
		size, size, b.String(), color, opts.StrokeWidth,
	)
}

// Subsample returns a subset of points such that consecutive points are at
// least minDistM meters apart. The first and last points are always included.
func (pts Points) Subsample(minDistM float64) Points {
	if len(pts) <= 2 {
		return pts
	}
	result := Points{pts[0]}
	last := &pts[0]
	for i := 1; i < len(pts)-1; i++ {
		if last.MetersTo(&pts[i]) >= minDistM {
			result = append(result, pts[i])
			last = &pts[i]
		}
	}
	result = append(result, pts[len(pts)-1])
	return result
}
