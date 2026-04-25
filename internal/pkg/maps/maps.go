package maps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/protomaps/go-pmtiles/pmtiles"
	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/client"
)

// BuildsURL is the URL for the protomaps build metadata index.
const BuildsURL = "https://build-metadata.protomaps.dev/builds.json"

// DataAttribution is the TASL attribution for OpenStreetMap map tile data served via Protomaps.
var DataAttribution = attribute.Attribution{
	What:       "Protomaps",
	Title:      "OpenStreetMap",
	Author:     "OpenStreetMap contributors",
	Source:     "https://www.openstreetmap.org/copyright",
	License:    "ODbL",
	LicenseURL: "https://opendatacommons.org/licenses/odbl/",
}

// BuildMetadata represents a single entry in the protomaps builds.json file.
type BuildMetadata struct {
	Key      string    `json:"key"`
	Size     int64     `json:"size"`
	MD5Sum   string    `json:"md5sum"`
	B3Sum    string    `json:"b3sum"`
	Uploaded time.Time `json:"uploaded"`
	Version  string    `json:"version"`
}

// FetchBuilds downloads and parses the builds.json index from protomaps.
func FetchBuilds(ctx context.Context) ([]BuildMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, BuildsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.New().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch builds.json: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching builds.json", resp.StatusCode)
	}

	var builds []BuildMetadata
	if err := json.NewDecoder(resp.Body).Decode(&builds); err != nil {
		return nil, fmt.Errorf("decode builds.json: %w", err)
	}

	return builds, nil
}

// LatestBuild returns the most recently uploaded build from the list.
// Returns an error if the list is empty.
func LatestBuild(builds []BuildMetadata) (BuildMetadata, error) {
	if len(builds) == 0 {
		return BuildMetadata{}, fmt.Errorf("no builds available")
	}

	latest := builds[0]
	for _, b := range builds[1:] {
		if b.Uploaded.After(latest.Uploaded) {
			latest = b
		}
	}
	return latest, nil
}

// ExtractParams holds the parameters for a PMTiles extraction.
type ExtractParams struct {
	// BucketURL is the base URL for the PMTiles source bucket.
	BucketURL string
	// Key is the filename within the bucket (e.g. "20260411.pmtiles").
	Key string
	// MaxZoom is the maximum zoom level to extract.
	MaxZoom int
	// Bbox is the bounding box string (min_lon,min_lat,max_lon,max_lat).
	Bbox string
	// OutputPath is the local file path for the extracted archive.
	OutputPath string
}

// SourceBucketURL is the base URL for the protomaps PMTiles archive bucket.
const SourceBucketURL = "https://build.protomaps.com"

// Extract downloads a regional extract from a remote PMTiles archive.
func Extract(ctx context.Context, params ExtractParams) error {
	logger := log.New(io.Discard, "", 0)

	dir := filepath.Dir(params.OutputPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	return pmtiles.Extract(
		ctx,
		logger,
		params.BucketURL,
		params.Key,
		0,                    // minzoom
		int8(params.MaxZoom), // maxzoom
		"",                   // regionFile
		params.Bbox,          // bbox
		params.OutputPath,    // output
		4,                    // downloadThreads
		0.05,                 // overfetch
		false,                // dryRun
	)
}

// OutputPath returns the file path for a map build identified by its UUID.
func OutputPath(mapsDir, uuid string) string {
	return filepath.Join(mapsDir, uuid+".pmtiles")
}
