package maps

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/jobs"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

const (
	// jobTimeout is the maximum time the map download job may run.
	jobTimeout = 60 * time.Minute

	// minRefreshAge is the minimum age of the last successful build before a new one is attempted.
	minRefreshAge = 23 * time.Hour
)

// DownloaderArgs are the arguments for the map downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return "maps.downloader" }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader periodically extracts a regional PMTiles subset from the latest protomaps build.
// Use [NewDownloader] to create an instance.
type Downloader struct {
	d       *db.DB
	cfg     MapsConfig
	mapsDir string
	bbox    *Bbox
}

// NewDownloader creates a new [Downloader] instance.
// The mapsDir parameter is the directory where extracted PMTiles files are written.
// The config must have been validated before calling this function.
func NewDownloader(d *db.DB, cfg MapsConfig, mapsDir string) *Downloader {
	bbox, _ := cfg.ParsedBbox()
	return &Downloader{d: d, cfg: cfg, mapsDir: mapsDir, bbox: bbox}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// Run implements [jobs.Job].
// It fetches the current builds index, checks whether the latest build has already been
// extracted with the same parameters, and if not, runs a PMTiles extraction and records
// the result in the database.
func (dl *Downloader) Run(ctx context.Context, _ DownloaderArgs) error {
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	// Check if a recent ready build exists.
	latest, err := dl.d.QueryRO().GetLatestReadyMapBuild(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check latest build: %w", err)
	}
	if err == nil && time.Since(latest.CreatedAt) < minRefreshAge {
		logg.Info(ctx, "map data is recent, skipping download", "lastBuild", latest.CreatedAt)
		return nil
	}

	// Fetch build index.
	builds, err := FetchBuilds(ctx)
	if err != nil {
		return fmt.Errorf("fetch builds: %w", err)
	}

	build, err := LatestBuild(builds)
	if err != nil {
		return err
	}

	logg.Info(ctx, "latest protomaps build", "key", build.Key, "version", build.Version, "uploaded", build.Uploaded)

	// Check if this exact build+params combination was already extracted.
	lookupParams := db.GetMapBuildByKeyParams{
		Key:     build.Key,
		Maxzoom: int64(dl.cfg.MapsMaxZoom),
	}
	if dl.bbox != nil {
		lookupParams.BboxMinLon = dl.bbox.NullMinLon()
		lookupParams.BboxMinLat = dl.bbox.NullMinLat()
		lookupParams.BboxMaxLon = dl.bbox.NullMaxLon()
		lookupParams.BboxMaxLat = dl.bbox.NullMaxLat()
	}
	_, err = dl.d.QueryRO().GetMapBuildByKey(ctx, lookupParams)
	if err == nil {
		logg.Info(ctx, "build already extracted, skipping", "key", build.Key)
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check existing build: %w", err)
	}

	return dl.extractAndRecord(ctx, build)
}

// extractAndRecord performs the PMTiles extraction and records the result in the database.
func (dl *Downloader) extractAndRecord(ctx context.Context, build BuildMetadata) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate uuid: %w", err)
	}

	outPath := OutputPath(dl.mapsDir, id.String())

	// Insert the build record before extraction (ready=0).
	insertParams := db.InsertMapBuildParams{
		Uuid:      id.String(),
		CreatedAt: time.Now(),
		Key:       build.Key,
		Size:      build.Size,
		Md5sum:    build.MD5Sum,
		Uploaded:  build.Uploaded,
		Version:   build.Version,
		Maxzoom:   int64(dl.cfg.MapsMaxZoom),
	}
	if dl.bbox != nil {
		insertParams.BboxMinLon = dl.bbox.NullMinLon()
		insertParams.BboxMinLat = dl.bbox.NullMinLat()
		insertParams.BboxMaxLon = dl.bbox.NullMaxLon()
		insertParams.BboxMaxLat = dl.bbox.NullMaxLat()
	}
	if err := dl.d.QueryRW().InsertMapBuild(ctx, insertParams); err != nil {
		return fmt.Errorf("insert map build: %w", err)
	}

	bboxStr := ""
	if dl.bbox != nil {
		bboxStr = dl.bbox.String()
	}
	logg.Info(ctx, "starting PMTiles extraction", "key", build.Key, "output", outPath, "bbox", bboxStr, "maxzoom", dl.cfg.MapsMaxZoom)

	err = Extract(ctx, ExtractParams{
		BucketURL:  SourceBucketURL,
		Key:        build.Key,
		MaxZoom:    dl.cfg.MapsMaxZoom,
		Bbox:       bboxStr,
		OutputPath: outPath,
	})
	if err != nil {
		// Clean up the partial file and DB record on failure.
		os.Remove(outPath)
		if _, delErr := dl.d.QueryRW().DeleteMapBuild(ctx, id.String()); delErr != nil {
			logg.Error(ctx, "failed to clean up map build record after extraction failure", "err", delErr)
		}
		return fmt.Errorf("extract PMTiles: %w", err)
	}

	if _, err := dl.d.QueryRW().SetMapBuildReady(ctx, id.String()); err != nil {
		return fmt.Errorf("mark build ready: %w", err)
	}

	logg.Info(ctx, "PMTiles extraction complete", "key", build.Key, "uuid", id.String())
	return nil
}
