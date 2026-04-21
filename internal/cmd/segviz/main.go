// Command segviz loads GPX/FIT files from a directory, runs the segment
// extraction algorithm, and writes KML files of intermediate steps for
// manual inspection. It does not touch any database.
//
// Usage:
//
//	go run ./internal/cmd/segviz --dir ./data/testuploads-gpxonly --out ./data/segviz-out
//	go run ./internal/cmd/segviz --dir ./data/testuploads-minimal --out ./data/segviz-out
package main

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/twpayne/go-kml/v3"
	"github.com/uber/h3-go/v4"

	"jo-m.ch/go/cartomancer/internal/pkg/load"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/segment"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	args := parseArgs()

	logger := logg.New(logg.LoggConfig{LogPretty: true})
	ctx := logg.WithLogger(context.Background(), logger)

	files, err := findTrackFiles(args.dir)
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no .gpx or .fit files found in %s", args.dir)
	}
	logg.Info(ctx, "found track files", "count", len(files))

	if err := os.MkdirAll(args.out, 0o750); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Load all tracks and build H3 cells.
	trackCells, sources, skipped := loadTracks(ctx, files, args.resolution)
	logg.Info(ctx, "loaded tracks", "loaded", len(trackCells), "skipped", skipped)

	if len(trackCells) < args.minTracks {
		return fmt.Errorf("need at least %d tracks, got %d", args.minTracks, len(trackCells))
	}

	// Step 1: write input tracks as KML.
	if err := writeInputTracksKML(args.out, trackCells, sources); err != nil {
		return fmt.Errorf("writing input tracks KML: %w", err)
	}
	logg.Info(ctx, "wrote 01_input_tracks.kml")

	// Phase 1: detect junctions.
	t0 := time.Now()
	rawJunctions, edgeIndex := segment.DetectJunctions(trackCells)
	logg.Info(ctx, "phase 1: detected junctions", "count", len(rawJunctions), "duration", time.Since(t0).Round(time.Millisecond))

	if err := writeCellSetKML(args.out, "03_junctions_raw.kml", rawJunctions, color.NRGBA{R: 255, G: 100, B: 0, A: 255}); err != nil {
		return fmt.Errorf("writing raw junctions KML: %w", err)
	}
	logg.Info(ctx, "wrote 03_junctions_raw.kml", "junctions", len(rawJunctions))

	// Phase 2: refine junctions using GPS track points.
	rawLoader := makeFileRawPointLoader(sources)
	junctions := rawJunctions
	t1 := time.Now()
	refinedJunctions, err := segment.RefineJunctions(rawJunctions, trackCells, rawLoader, args.resolution)
	if err != nil {
		return fmt.Errorf("refining junctions: %w", err)
	}
	junctions = refinedJunctions
	logg.Info(ctx, "phase 2: refined junctions", "count", len(refinedJunctions), "duration", time.Since(t1).Round(time.Millisecond))

	if err := writeCellSetKML(args.out, "04_junctions_refined.kml", refinedJunctions, color.NRGBA{R: 0, G: 180, B: 255, A: 255}); err != nil {
		return fmt.Errorf("writing refined junctions KML: %w", err)
	}
	logg.Info(ctx, "wrote 04_junctions_refined.kml", "junctions", len(refinedJunctions))

	// Phase 3: slice tracks at junctions (with dedup).
	t2 := time.Now()
	rawSegments := segment.SliceAtJunctions(trackCells, junctions, edgeIndex)
	logg.Info(ctx, "phase 3: sliced segments", "raw", len(rawSegments), "duration", time.Since(t2).Round(time.Millisecond))

	if err := writeSegmentsKML(args.out, "05_segments_raw.kml", rawSegments, false); err != nil {
		return fmt.Errorf("writing raw segments KML: %w", err)
	}
	logg.Info(ctx, "wrote 05_segments_raw.kml")

	// Phase 4: filter segments.
	segments := segment.FilterSegments(rawSegments, args.minTracks, segment.MinSegmentDistanceM)
	logg.Info(ctx, "phase 4: filtered segments", "kept", len(segments), "dropped", len(rawSegments)-len(segments))

	extractDur := time.Since(t0)

	// Attach polylines from actual GPS points.
	pointLoader := makeFilePointLoader(sources)
	if err := segment.AttachPolylines(segments, pointLoader, args.resolution); err != nil {
		return fmt.Errorf("attaching polylines: %w", err)
	}

	if err := writeSegmentsKML(args.out, "06_segments_final.kml", segments, true); err != nil {
		return fmt.Errorf("writing final segments KML: %w", err)
	}
	logg.Info(ctx, "wrote 06_segments_final.kml")

	nJunctions, err := writeJunctionsKML(args.out, segments)
	if err != nil {
		return fmt.Errorf("writing junctions KML: %w", err)
	}
	logg.Info(ctx, "wrote 07_junctions_segments.kml", "junctions", nJunctions)

	if err := writeStats(args, trackCells, segments, rawSegments, skipped, nJunctions, extractDur); err != nil {
		return fmt.Errorf("writing stats: %w", err)
	}
	logg.Info(ctx, "wrote 08_stats.txt")

	logg.Info(ctx, "done", "segments", len(segments), "outputDir", args.out)
	return nil
}

type cliArgs struct {
	dir        string
	out        string
	resolution int
	minTracks  int
}

func parseArgs() cliArgs {
	args := cliArgs{
		resolution: segment.DefaultResolution,
		minTracks:  segment.MinTrackCount,
	}

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--dir":
			i++
			args.dir = os.Args[i]
		case "--out":
			i++
			args.out = os.Args[i]
		case "--help", "-h":
			fmt.Fprintf(os.Stderr, "Usage: segviz --dir <tracks-dir> [--out <output-dir>]\n")
			fmt.Fprintf(os.Stderr, "\nLoads GPX/FIT files, runs segment extraction, writes KML files.\n")
			os.Exit(0)
		default:
			log.Fatalf("unknown argument: %s", os.Args[i])
		}
	}

	if args.dir == "" {
		log.Fatal("--dir is required")
	}
	if args.out == "" {
		args.out = filepath.Join(args.dir, "segviz-out")
	}
	return args
}

// findTrackFiles returns all .gpx and .fit files in the given directory.
func findTrackFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".gpx" || ext == ".fit" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

// loadTracks loads all track files and returns both TrackCells (for the
// segmenting algorithm) and the parsed TrackSources (for point loading).
// The third return value is the number of files that were skipped.
func loadTracks(ctx context.Context, files []string, resolution int) ([]segment.TrackCells, map[string]track.TrackSource, int) {
	var trackCells []segment.TrackCells
	sources := make(map[string]track.TrackSource)
	skipped := 0

	for _, path := range files {
		name := filepath.Base(path)
		src, err := load.Path(path)
		if err != nil {
			logg.Debug(ctx, "skipping file", "file", name, "err", err)
			skipped++
			continue
		}

		cells, err := track.NewCells(src, resolution)
		if err != nil {
			logg.Debug(ctx, "skipping file, cell build failed", "file", name, "err", err)
			skipped++
			continue
		}

		// Re-parse for the source cache since iterators are single-use.
		src2, err := load.Path(path)
		if err != nil {
			logg.Debug(ctx, "skipping file, re-parse failed", "file", name, "err", err)
			skipped++
			continue
		}

		sources[name] = src2
		trackCells = append(trackCells, segment.TrackCells{UUID: name, Cells: cells})
	}
	return trackCells, sources, skipped
}

// makeFilePointLoader builds a PointLoader that reads from the in-memory
// TrackSource map instead of a database.
// makeFileRawPointLoader returns a [segment.RawPointLoader] that returns all
// GPS points from in-memory track sources without cell indexing.
func makeFileRawPointLoader(sources map[string]track.TrackSource) segment.RawPointLoader {
	return func(uuid string) ([]track.Point, error) {
		src, ok := sources[uuid]
		if !ok {
			return nil, fmt.Errorf("track %q not found in loaded sources", uuid)
		}
		var pts []track.Point
		for p := range src.All() {
			pts = append(pts, p)
		}
		return pts, nil
	}
}

// makeFilePointLoader returns a [segment.PointLoader] that maps GPS points
// from in-memory track sources to H3 cells on demand. Results are cached
// by (uuid, resolution) to support multiple resolutions efficiently.
func makeFilePointLoader(sources map[string]track.TrackSource) segment.PointLoader {
	type cacheKey struct {
		uuid string
		res  int
	}
	cache := make(map[cacheKey]map[h3.Cell]track.Point)

	return func(uuid string, res int) (map[h3.Cell]track.Point, error) {
		key := cacheKey{uuid, res}
		if m, ok := cache[key]; ok {
			return m, nil
		}
		src, ok := sources[uuid]
		if !ok {
			return nil, fmt.Errorf("track %q not found in loaded sources", uuid)
		}
		m := make(map[h3.Cell]track.Point)
		for p := range src.All() {
			c := p.Cell(res)
			if _, exists := m[c]; !exists {
				m[c] = p
			}
		}
		cache[key] = m
		return m, nil
	}
}

// vibrantColor returns a vibrant color for the given index by sampling HSV
// space with high saturation (0.85) and value (0.95). The golden ratio is
// used to step through hues for maximum visual distinction between
// consecutive indices.
func vibrantColor(i int) color.Color {
	// Golden ratio conjugate for well-distributed hues.
	h := math.Mod(float64(i)*0.618033988749895, 1.0)
	r, g, b := hsvToRGB(h, 0.85, 0.95)
	return color.NRGBA{R: r, G: g, B: b, A: 220}
}

// hsvToRGB converts HSV (h in [0,1], s in [0,1], v in [0,1]) to RGB bytes.
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	h *= 6
	i := math.Floor(h)
	f := h - i
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))

	var r, g, b float64
	switch int(i) % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	case 5:
		r, g, b = v, p, q
	}
	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}

// writeInputTracksKML writes the original GPS tracks as KML line strings.
func writeInputTracksKML(outDir string, trackCells []segment.TrackCells, sources map[string]track.TrackSource) error {
	var placemarks []kml.Element
	for i, tc := range trackCells {
		src, ok := sources[tc.UUID]
		if !ok {
			continue
		}

		var coords []kml.Coordinate
		for p := range src.All() {
			coords = append(coords, kml.Coordinate{Lon: p.Lon, Lat: p.Lat, Alt: p.Elevation})
		}
		if len(coords) < 2 {
			continue
		}

		placemark := kml.Placemark(
			kml.Name(tc.UUID),
			kml.Style(
				kml.LineStyle(
					kml.Color(vibrantColor(i)),
					kml.Width(3),
				),
			),
			kml.LineString(
				kml.Coordinates(coords...),
				kml.Tessellate(true),
			),
		)
		placemarks = append(placemarks, placemark)
	}

	doc := kml.KML(
		kml.Document(
			append([]kml.Element{kml.Name("Input Tracks")}, placemarks...)...,
		),
	)
	return writeKMLFile(filepath.Join(outDir, "01_input_tracks.kml"), doc)
}

// writeSegmentsKML writes extracted segments as KML. When usePolyline is true,
// it uses the attached GPS polyline; otherwise it uses H3 cell centers.
func writeSegmentsKML(outDir, filename string, segments []segment.Segment, usePolyline bool) error {
	var placemarks []kml.Element
	for i, seg := range segments {
		var coords []kml.Coordinate

		if usePolyline && len(seg.Polyline) >= 2 {
			for _, pt := range seg.Polyline {
				coords = append(coords, kml.Coordinate{Lat: pt[0], Lon: pt[1]})
			}
		} else {
			for _, c := range seg.Cells {
				ll, err := c.LatLng()
				if err != nil {
					continue
				}
				coords = append(coords, kml.Coordinate{Lon: ll.Lng, Lat: ll.Lat})
			}
		}

		if len(coords) < 2 {
			continue
		}

		desc := fmt.Sprintf("tracks: %d, distance: %.0f m, cells: %d",
			len(seg.TrackUUIDs), seg.DistanceM, len(seg.Cells))

		c := vibrantColor(i)
		placemark := kml.Placemark(
			kml.Description(desc),
			kml.Style(
				kml.LineStyle(
					kml.Color(c),
					kml.Width(4),
				),
			),
			kml.LineString(
				kml.Coordinates(coords...),
				kml.Tessellate(true),
			),
		)
		placemarks = append(placemarks, placemark)

		if arrow := arrowPlacemark(coords, c); arrow != nil {
			placemarks = append(placemarks, arrow)
		}
	}

	doc := kml.KML(
		kml.Document(
			append([]kml.Element{kml.Name(filename)}, placemarks...)...,
		),
	)
	return writeKMLFile(filepath.Join(outDir, filename), doc)
}

// arrowPlacemark creates a small chevron (">" shape) at the end of a
// coordinate sequence indicating the direction of travel.
func arrowPlacemark(coords []kml.Coordinate, c color.Color) kml.Element {
	n := len(coords)
	if n < 2 {
		return nil
	}

	tip := coords[n-1]
	prev := coords[n-2]

	cosLat := math.Cos(tip.Lat * math.Pi / 180)
	if cosLat < 1e-6 {
		return nil
	}

	// Direction in approximate meters.
	const mPerDeg = 111_000.0
	dxm := (tip.Lon - prev.Lon) * cosLat * mPerDeg
	dym := (tip.Lat - prev.Lat) * mPerDeg
	length := math.Sqrt(dxm*dxm + dym*dym)
	if length < 0.1 {
		return nil
	}
	dxm /= length
	dym /= length

	// Wing length in meters.
	const wingLen = 60.0
	const cos150 = -0.866025
	const sin150 = 0.5

	w1xm := (dxm*cos150 - dym*sin150) * wingLen
	w1ym := (dxm*sin150 + dym*cos150) * wingLen

	w2xm := (dxm*cos150 + dym*sin150) * wingLen
	w2ym := (-dxm*sin150 + dym*cos150) * wingLen

	w1 := kml.Coordinate{
		Lat: tip.Lat + w1ym/mPerDeg,
		Lon: tip.Lon + w1xm/(cosLat*mPerDeg),
	}
	w2 := kml.Coordinate{
		Lat: tip.Lat + w2ym/mPerDeg,
		Lon: tip.Lon + w2xm/(cosLat*mPerDeg),
	}

	return kml.Placemark(
		kml.Style(kml.LineStyle(kml.Color(c), kml.Width(3))),
		kml.LineString(
			kml.Coordinates(w1, tip, w2),
			kml.Tessellate(true),
		),
	)
}

// writeJunctionsKML writes all unique junctions as KML point placemarks.
// Returns the number of unique junctions written.
func writeJunctionsKML(outDir string, segments []segment.Segment) (int, error) {
	type junctionInfo struct {
		lat, lon float64
		degree   int // number of segments meeting at this junction
	}
	seen := make(map[h3.Cell]*junctionInfo)

	for _, seg := range segments {
		for _, j := range []segment.Junction{seg.StartJunction, seg.EndJunction} {
			if info, ok := seen[j.H3Cell]; ok {
				info.degree++
			} else {
				seen[j.H3Cell] = &junctionInfo{lat: j.Lat, lon: j.Lon, degree: 1}
			}
		}
	}

	junctionColor := color.NRGBA{R: 0, G: 220, B: 80, A: 255}
	strokeColor := color.NRGBA{R: 0, G: 220, B: 80, A: 200}
	fillColor := color.NRGBA{R: 0, G: 220, B: 80, A: 80}

	var placemarks []kml.Element
	for cell, info := range seen {
		desc := fmt.Sprintf("H3 cell: %s\ndegree: %d", cell, info.degree)

		// Point placemark for the junction location.
		placemarks = append(placemarks, kml.Placemark(
			kml.Name(fmt.Sprintf("junction_%s", cell)),
			kml.Description(desc),
			kml.Style(
				kml.IconStyle(
					kml.Color(junctionColor),
					kml.Scale(0.8),
				),
			),
			kml.Point(
				kml.Coordinates(kml.Coordinate{Lat: info.lat, Lon: info.lon}),
			),
		))

		// Polygon placemark showing the H3 cell boundary.
		boundary, err := cell.Boundary()
		if err != nil || len(boundary) < 3 {
			continue
		}
		ringCoords := make([]kml.Coordinate, 0, len(boundary)+1)
		for _, ll := range boundary {
			ringCoords = append(ringCoords, kml.Coordinate{Lat: ll.Lat, Lon: ll.Lng})
		}
		ringCoords = append(ringCoords, ringCoords[0]) // Close the ring.

		placemarks = append(placemarks, kml.Placemark(
			kml.Name(fmt.Sprintf("cell_%s", cell)),
			kml.Description(desc),
			kml.Style(
				kml.LineStyle(
					kml.Color(strokeColor),
					kml.Width(2),
				),
				kml.PolyStyle(
					kml.Color(fillColor),
				),
			),
			kml.Polygon(
				kml.OuterBoundaryIs(
					kml.LinearRing(
						kml.Coordinates(ringCoords...),
					),
				),
				kml.Tessellate(true),
			),
		))
	}

	doc := kml.KML(
		kml.Document(
			append([]kml.Element{kml.Name("Junctions")}, placemarks...)...,
		),
	)
	return len(seen), writeKMLFile(filepath.Join(outDir, "07_junctions_segments.kml"), doc)
}

// writeCellSetKML writes a set of H3 cells as point placemarks to a KML file.
// The c parameter controls the color used for icons and cell boundary polygons.
func writeCellSetKML(outDir, filename string, cells map[h3.Cell]struct{}, c color.NRGBA) error {
	fillColor := color.NRGBA{R: c.R, G: c.G, B: c.B, A: 80}
	strokeColor := color.NRGBA{R: c.R, G: c.G, B: c.B, A: 200}

	placemarks := make([]kml.Element, 0, len(cells)*2)
	for cell := range cells {
		ll, err := cell.LatLng()
		if err != nil {
			continue
		}
		placemarks = append(placemarks, kml.Placemark(
			kml.Style(
				kml.IconStyle(
					kml.Color(c),
					kml.Scale(0.5),
				),
			),
			kml.Point(
				kml.Coordinates(kml.Coordinate{Lat: ll.Lat, Lon: ll.Lng}),
			),
		))

		boundary, bErr := cell.Boundary()
		if bErr != nil || len(boundary) < 3 {
			continue
		}
		ringCoords := make([]kml.Coordinate, 0, len(boundary)+1)
		for _, bll := range boundary {
			ringCoords = append(ringCoords, kml.Coordinate{Lat: bll.Lat, Lon: bll.Lng})
		}
		ringCoords = append(ringCoords, ringCoords[0])

		placemarks = append(placemarks, kml.Placemark(
			kml.Style(
				kml.LineStyle(
					kml.Color(strokeColor),
					kml.Width(2),
				),
				kml.PolyStyle(
					kml.Color(fillColor),
				),
			),
			kml.Polygon(
				kml.OuterBoundaryIs(
					kml.LinearRing(
						kml.Coordinates(ringCoords...),
					),
				),
				kml.Tessellate(true),
			),
		))
	}

	doc := kml.KML(
		kml.Document(
			append([]kml.Element{kml.Name(filename)}, placemarks...)...,
		),
	)
	return writeKMLFile(filepath.Join(outDir, filename), doc)
}

func writeKMLFile(path string, doc *kml.KMLElement) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := doc.WriteIndent(f, "", "  "); err != nil {
		return err
	}
	return f.Close()
}

// writeStats writes a human-readable summary of the extraction run.
func writeStats(args cliArgs, trackCells []segment.TrackCells, segments []segment.Segment, rawSegments []segment.Segment, skipped, nJunctions int, extractDur time.Duration) error {
	totalCells := 0
	for _, tc := range trackCells {
		totalCells += tc.Cells.NCells()
	}

	totalDistM := 0.0
	minDistM := 0.0
	maxDistM := 0.0
	totalSegCells := 0
	maxTracks := 0
	trackCountHist := make(map[int]int)
	for i, seg := range segments {
		totalDistM += seg.DistanceM
		totalSegCells += len(seg.Cells)
		if i == 0 || seg.DistanceM < minDistM {
			minDistM = seg.DistanceM
		}
		if seg.DistanceM > maxDistM {
			maxDistM = seg.DistanceM
		}
		n := len(seg.TrackUUIDs)
		trackCountHist[n]++
		if n > maxTracks {
			maxTracks = n
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "segviz extraction stats\n")
	fmt.Fprintf(&b, "=======================\n\n")

	fmt.Fprintf(&b, "input\n")
	fmt.Fprintf(&b, "  directory:      %s\n", args.dir)
	fmt.Fprintf(&b, "  tracks loaded:  %d\n", len(trackCells))
	fmt.Fprintf(&b, "  tracks skipped: %d\n", skipped)
	fmt.Fprintf(&b, "  total H3 cells: %d\n", totalCells)
	fmt.Fprintf(&b, "  H3 resolution:  %d\n\n", args.resolution)

	fmt.Fprintf(&b, "extraction\n")
	fmt.Fprintf(&b, "  min tracks:     %d\n", args.minTracks)
	fmt.Fprintf(&b, "  duration:       %s\n", extractDur.Round(time.Millisecond))
	fmt.Fprintf(&b, "  raw segments:   %d\n", len(rawSegments))
	fmt.Fprintf(&b, "  segments:       %d\n", len(segments))
	fmt.Fprintf(&b, "  junctions:      %d\n\n", nJunctions)

	if len(segments) > 0 {
		avgDistM := totalDistM / float64(len(segments))
		fmt.Fprintf(&b, "segment distances\n")
		fmt.Fprintf(&b, "  total:   %.0f m (%.1f km)\n", totalDistM, totalDistM/1000)
		fmt.Fprintf(&b, "  min:     %.0f m\n", minDistM)
		fmt.Fprintf(&b, "  max:     %.0f m (%.1f km)\n", maxDistM, maxDistM/1000)
		fmt.Fprintf(&b, "  average: %.0f m\n", avgDistM)
		fmt.Fprintf(&b, "  total cells: %d\n\n", totalSegCells)

		fmt.Fprintf(&b, "track count distribution (how many tracks share each segment)\n")
		counts := make([]int, 0, len(trackCountHist))
		for k := range trackCountHist {
			counts = append(counts, k)
		}
		slices.Sort(counts)
		for _, n := range counts {
			fmt.Fprintf(&b, "  %3d tracks: %d segments\n", n, trackCountHist[n])
		}
	}

	return os.WriteFile(filepath.Join(args.out, "06_stats.txt"), []byte(b.String()), 0o600)
}
