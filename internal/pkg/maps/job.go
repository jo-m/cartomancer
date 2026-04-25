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
	minRefreshAge = 70 * time.Hour
)

// DownloaderArgs are the arguments for the map downloader job.
type DownloaderArgs struct{}

// Kind implements [jobs.Args].
func (DownloaderArgs) Kind() string { return "maps.downloader" }

var _ jobs.Args = (*DownloaderArgs)(nil)

// Downloader periodically extracts regional PMTiles subsets from the latest protomaps build.
// Use [NewDownloader] to create an instance.
type Downloader struct {
	d       *db.DB
	specs   []MapSpec
	mapsDir string
}

// NewDownloader creates a new [Downloader] instance.
// The mapsDir parameter is the directory where extracted PMTiles files are written.
// The config must have been validated before calling this function.
func NewDownloader(d *db.DB, cfg MapsConfig, mapsDir string) *Downloader {
	specs, _ := cfg.ParsedSpecs()
	return &Downloader{d: d, specs: specs, mapsDir: mapsDir}
}

var _ jobs.Job[DownloaderArgs] = (*Downloader)(nil)

// Run implements [jobs.Job].
// It fetches the current builds index and performs an extraction for each configured spec.
// Each extraction is independently deduplicated.
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

	for i, spec := range dl.specs {
		if err := dl.ensureExtract(ctx, build, spec); err != nil {
			return fmt.Errorf("spec %d: %w", i, err)
		}
	}

	return nil
}

// ensureExtract checks whether the given build+spec combination already exists
// and performs the extraction if not. The existence check and the initial DB
// record insertion are wrapped in a single transaction to prevent TOCTOU races
// when jobs run concurrently.
func (dl *Downloader) ensureExtract(ctx context.Context, build BuildMetadata, spec MapSpec) error {
	lookupParams := db.GetMapBuildByKeyParams{
		Key:     build.Key,
		Maxzoom: int64(spec.MaxZoom),
	}
	if spec.Bbox != nil {
		lookupParams.BboxMinLon = spec.Bbox.NullMinLon()
		lookupParams.BboxMinLat = spec.Bbox.NullMinLat()
		lookupParams.BboxMaxLon = spec.Bbox.NullMaxLon()
		lookupParams.BboxMaxLat = spec.Bbox.NullMaxLat()
	}

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate uuid: %w", err)
	}

	insertParams := db.InsertMapBuildParams{
		Uuid:      id.String(),
		CreatedAt: time.Now(),
		Key:       build.Key,
		Size:      build.Size,
		Md5sum:    build.MD5Sum,
		Uploaded:  build.Uploaded,
		Version:   build.Version,
		Maxzoom:   int64(spec.MaxZoom),
	}
	if spec.Bbox != nil {
		insertParams.BboxMinLon = spec.Bbox.NullMinLon()
		insertParams.BboxMinLat = spec.Bbox.NullMinLat()
		insertParams.BboxMaxLon = spec.Bbox.NullMaxLon()
		insertParams.BboxMaxLat = spec.Bbox.NullMaxLat()
	}

	var alreadyExists bool
	err = dl.d.WithTx(ctx, func(tx *db.Queries) error {
		_, txErr := tx.GetMapBuildByKey(ctx, lookupParams)
		if txErr == nil {
			alreadyExists = true
			return nil
		}
		if !errors.Is(txErr, sql.ErrNoRows) {
			return fmt.Errorf("check existing build: %w", txErr)
		}
		return tx.InsertMapBuild(ctx, insertParams)
	})
	if err != nil {
		return err
	}
	if alreadyExists {
		logg.Info(ctx, "build already extracted, skipping", "key", build.Key, "maxzoom", spec.MaxZoom)
		return nil
	}

	return dl.extractAndRecord(ctx, id, OutputPath(dl.mapsDir, id.String()), build, spec)
}

// extractAndRecord performs the PMTiles extraction for a build record that has
// already been inserted into the database with the given id.
func (dl *Downloader) extractAndRecord(ctx context.Context, id uuid.UUID, outPath string, build BuildMetadata, spec MapSpec) error {

	bboxStr := ""
	if spec.Bbox != nil {
		bboxStr = spec.Bbox.String()
	}
	logg.Info(ctx, "starting PMTiles extraction", "key", build.Key, "output", outPath, "bbox", bboxStr, "maxzoom", spec.MaxZoom)

	err := Extract(ctx, ExtractParams{
		BucketURL:  SourceBucketURL,
		Key:        build.Key,
		MaxZoom:    spec.MaxZoom,
		Bbox:       bboxStr,
		OutputPath: outPath,
	})
	if err != nil {
		if rmErr := os.Remove(outPath); rmErr != nil && !os.IsNotExist(rmErr) {
			logg.Error(ctx, "failed to remove output file after extraction failure", "err", rmErr)
		}
		if _, delErr := dl.d.QueryRW().DeleteMapBuild(ctx, id.String()); delErr != nil {
			logg.Error(ctx, "failed to clean up map build record after extraction failure", "err", delErr)
		}
		return fmt.Errorf("extract PMTiles: %w", err)
	}

	if _, err := dl.d.QueryRW().SetMapBuildReady(ctx, id.String()); err != nil {
		return fmt.Errorf("mark build ready: %w", err)
	}

	if info, statErr := os.Stat(outPath); statErr == nil {
		if err := dl.d.QueryRW().SetMapBuildLocalSize(ctx, db.SetMapBuildLocalSizeParams{
			LocalSize: sql.NullInt64{Valid: true, Int64: info.Size()},
			Uuid:      id.String(),
		}); err != nil {
			logg.Error(ctx, "failed to record local map file size", "err", err)
		}
	}

	markParams := db.MarkOlderMapBuildsForDeletionParams{
		Uuid:    id.String(),
		Maxzoom: int64(spec.MaxZoom),
	}
	if spec.Bbox != nil {
		markParams.BboxMinLon = spec.Bbox.NullMinLon()
		markParams.BboxMinLat = spec.Bbox.NullMinLat()
		markParams.BboxMaxLon = spec.Bbox.NullMaxLon()
		markParams.BboxMaxLat = spec.Bbox.NullMaxLat()
	}
	if _, err := dl.d.QueryRW().MarkOlderMapBuildsForDeletion(ctx, markParams); err != nil {
		logg.Error(ctx, "failed to mark older map builds for deletion", "err", err)
	}

	logg.Info(ctx, "PMTiles extraction complete", "key", build.Key, "uuid", id.String())
	return nil
}
