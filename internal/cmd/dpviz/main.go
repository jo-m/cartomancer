// Command dpviz loads a GPX or FIT track file, applies Douglas-Peucker
// simplification at several epsilon values (2, 5, 10, 20, 50, 100, 200 m),
// and writes the simplified GPX file, the encoded polyline as a .txt file,
// and the varint-encoded points as a .bin file for each epsilon to a
// subdirectory named after the input file.
//
// Usage:
//
//	go run ./internal/cmd/dpviz mytrack.gpx
//	go run ./internal/cmd/dpviz myactivity.fit
package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"jo-m.ch/go/cartomancer/internal/pkg/load"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

var epsilons = []float64{5, 10, 50, 100}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: dpviz <file.gpx|file.fit>\n")
		os.Exit(1)
	}

	srcPath := os.Args[1]

	src, err := load.Path(srcPath)
	if err != nil {
		return fmt.Errorf("loading %s: %w", srcPath, err)
	}

	var pts track.Points
	for p := range src.All() {
		pts = append(pts, p)
	}

	base := filepath.Base(srcPath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	if err := os.MkdirAll(stem, 0o750); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	origPolylineBytes := len(track.EncodePolyline(pts))
	origVarint, err := track.EncodeVarint(pts)
	if err != nil {
		return fmt.Errorf("varint encoding original: %w", err)
	}
	origInt32Bytes := len(pts) * 2 * 4
	fmt.Printf("original:          %5d pts  polyline=%6d B  varint=%6d B  2xint32=%6d B\n",
		len(pts), origPolylineBytes, len(origVarint), origInt32Bytes)

	for _, eps := range epsilons {
		simplified := pts.SimplifyDP(eps)

		name := fmt.Sprintf("simplified_%dm", int(eps))
		gpxFile := filepath.Join(stem, name+".gpx")
		txtFile := filepath.Join(stem, name+".txt")
		binFile := filepath.Join(stem, name+".bin")

		if err := writeGPX(gpxFile, simplified); err != nil {
			return fmt.Errorf("writing %s: %w", gpxFile, err)
		}

		polyline := track.EncodePolyline(simplified)
		if err := os.WriteFile(txtFile, []byte(polyline), 0o600); err != nil {
			return fmt.Errorf("writing %s: %w", txtFile, err)
		}

		varintBuf, err := track.EncodeVarint(simplified)
		if err != nil {
			return fmt.Errorf("varint encoding %s: %w", name, err)
		}
		if err := os.WriteFile(binFile, varintBuf, 0o600); err != nil {
			return fmt.Errorf("writing %s: %w", binFile, err)
		}

		nPts := len(simplified)
		polylineBytes := len(polyline)
		int32Bytes := nPts * 2 * 4
		fmt.Printf("epsilon=%3dm: %5d pts  polyline=%6d B  varint=%6d B  2xint32=%6d B  -> %s\n",
			int(eps), nPts, polylineBytes, len(varintBuf), int32Bytes, gpxFile)
	}

	return nil
}

// gpxOutput is the root element for a written GPX file.
type gpxOutput struct {
	XMLName xml.Name       `xml:"gpx"`
	Version string         `xml:"version,attr"`
	Creator string         `xml:"creator,attr"`
	Track   gpxTrackOutput `xml:"trk"`
}

// gpxTrackOutput holds a single track element.
type gpxTrackOutput struct {
	Segment gpxSegmentOutput `xml:"trkseg"`
}

// gpxSegmentOutput holds a sequence of track points.
type gpxSegmentOutput struct {
	Points []gpxPointOutput `xml:"trkpt"`
}

// gpxPointOutput represents a single trkpt element.
type gpxPointOutput struct {
	Lat float64 `xml:"lat,attr"`
	Lon float64 `xml:"lon,attr"`
	Ele float64 `xml:"ele"`
}

// writeGPX encodes pts as a GPX 1.1 file at path.
func writeGPX(path string, pts track.Points) error {
	out := gpxOutput{
		Version: "1.1",
		Creator: "dpviz",
	}
	out.Track.Segment.Points = make([]gpxPointOutput, len(pts))
	for i, p := range pts {
		out.Track.Segment.Points[i] = gpxPointOutput{
			Lat: p.Lat,
			Lon: p.Lon,
			Ele: p.Elevation,
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	return f.Close()
}
