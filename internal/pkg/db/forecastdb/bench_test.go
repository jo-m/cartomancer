package forecastdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

// The forecastdb benchmarks model a single ICON-CH1-EPS forecast run as it
// arrives in production:
//
//   - 33 hourly steps per forecast (the model's published horizon).
//   - 5 variables per step, each a 2 MiB GRIB2 file (matching the fixed size
//     of real production GRIB2 payloads).
//   - Consecutive forecasts are produced every 3 hours.
//
// One forecast therefore holds 33 * 5 = 165 file rows totalling 330 MiB of
// blob payload. The listing and deletion benchmarks seed 10 such forecasts
// (1650 rows, ~3.3 GiB on disk). Run with -benchtime=1x for realistic
// single-shot measurements:
//
//	go test -bench=. -benchtime=1x ./internal/pkg/db/forecastdb/

// realisticBlobSize is the payload size of every simulated GRIB2 file. In
// production all ICON-CH1-EPS files for the variables we ingest are exactly
// this size.
const realisticBlobSize = 2252 * 1024

// realisticVars lists the 5 variables included in each hourly step of a
// simulated forecast. Every file is [realisticBlobSize] bytes.
var realisticVars = []string{"T_2M", "U_10M", "V_10M", "TOT_PREC", "CLCT"}

const (
	// realisticForecastSteps is the number of hourly forecast files per
	// variable per forecast, matching the ICON-CH1-EPS 33 h horizon.
	realisticForecastSteps = 33
	// realisticNumForecasts is the number of consecutive forecasts seeded
	// for the listing and deletion benchmarks.
	realisticNumForecasts = 10
	// realisticForecastSpacing is the cadence at which consecutive forecast
	// runs are produced.
	realisticForecastSpacing = 3 * time.Hour
)

// makeBlob returns a [realisticBlobSize]-byte buffer filled with a non-zero
// pattern so SQLite cannot apply any zero-fill optimisation that would
// understate the real on-disk cost. The same buffer is reused across every
// insert in the bench; the SQLite driver copies on bind.
func makeBlob() []byte {
	buf := make([]byte, realisticBlobSize)
	for i := range buf {
		buf[i] = byte(i)
	}
	return buf
}

// totalForecastBytes returns the on-disk blob payload size of one simulated
// forecast (33 steps * 5 variables * [realisticBlobSize]).
func totalForecastBytes() int64 {
	return int64(realisticBlobSize) * int64(len(realisticVars)) * realisticForecastSteps
}

// totalForecastFiles returns the number of forecast_files rows in one
// simulated forecast.
func totalForecastFiles() int {
	return realisticForecastSteps * len(realisticVars)
}

// openBenchDB creates a fresh forecast database under b.TempDir() with the
// production pragmas applied (32 KiB pages, WAL, incremental auto-vacuum).
// The caller must Close the returned DB.
func openBenchDB(b *testing.B) *DB {
	b.Helper()
	dir := b.TempDir()
	ctx := logg.WithDiscardHandler(context.Background())
	d, err := Open(ctx, filepath.Join(dir, "forecast.db"))
	require.NoError(b, err)
	return d
}

// seedRealisticForecast inserts a parent forecast row plus all 33 hourly
// steps, each step carrying one file per variable in [realisticVars]. Uses
// a single transaction to match the production ingestion path in
// meteo/job.go.
func seedRealisticForecast(b *testing.B, d *DB, refTime time.Time, blob []byte) {
	b.Helper()
	ctx := logg.WithDiscardHandler(context.Background())
	fc, err := d.QueryRW().CreateForecast(ctx, CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		BoundsMinLat:       sql.NullFloat64{Float64: 43.0, Valid: true},
		BoundsMinLon:       sql.NullFloat64{Float64: 2.0, Valid: true},
		BoundsMaxLat:       sql.NullFloat64{Float64: 50.0, Valid: true},
		BoundsMaxLon:       sql.NullFloat64{Float64: 16.0, Valid: true},
		HorizontalGridFile: []byte("hgrid"),
		VerticalGridFile:   []byte("vgrid"),
	})
	require.NoError(b, err)

	require.NoError(b, d.WithTx(ctx, func(tx *Queries) error {
		for step := range realisticForecastSteps {
			vt := refTime.Add(time.Duration(step) * time.Hour)
			for _, name := range realisticVars {
				if _, err := tx.CreateForecastFile(ctx, CreateForecastFileParams{
					ValidTime:      vt,
					ValidUntilTime: vt.Add(time.Hour),
					Variable:       name,
					File:           blob,
					ForecastID:     fc.ID,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	}))
}

// BenchmarkInsertForecast measures the cost of ingesting one complete
// forecast (33 hourly steps * 5 variables, ~5.4 GiB of blob payload)
// through the same CreateForecastFile helper and single-transaction pattern
// used by the meteo ingestion job. Each iteration uses a distinct
// reference_time so the UNIQUE constraint on forecasts.reference_time is
// never violated.
func BenchmarkInsertForecast(b *testing.B) {
	d := openBenchDB(b)
	defer d.Close()
	blob := makeBlob()

	base := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	b.SetBytes(totalForecastBytes())
	b.ResetTimer()

	for i := range b.N {
		refTime := base.Add(time.Duration(i) * realisticForecastSpacing)
		seedRealisticForecast(b, d, refTime, blob)
	}
}

// BenchmarkListForecastsWithFiles measures the admin listing query against
// a DB pre-populated with 10 consecutive forecasts (3 h apart), each holding
// 33 hourly steps * 5 variables = 165 files. With the split-blob schema and
// the file_size column on forecast_files, the query reads only the small
// metadata table and should be effectively constant in the blob payload
// size.
func BenchmarkListForecastsWithFiles(b *testing.B) {
	d := openBenchDB(b)
	defer d.Close()
	ctx := logg.WithDiscardHandler(context.Background())
	blob := makeBlob()

	base := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	for i := range realisticNumForecasts {
		seedRealisticForecast(b, d, base.Add(time.Duration(i)*realisticForecastSpacing), blob)
	}

	expectedRows := realisticNumForecasts * totalForecastFiles()
	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		rows, err := d.QueryRO().ListForecastsWithFiles(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if len(rows) != expectedRows {
			b.Fatalf("expected %d rows, got %d", expectedRows, len(rows))
		}
	}
}

// BenchmarkDeleteOutdatedForecastFiles measures the cost of the cleaner-path
// delete against 10 consecutive forecasts (3 h apart), each holding 33
// hourly steps * 5 variables = 165 files. The delete fans out through ON
// DELETE CASCADE into forecast_file_blobs, so the timer captures the full
// free-list walk over every overflow page. Each iteration re-seeds; setup
// time is excluded via StopTimer/StartTimer.
func BenchmarkDeleteOutdatedForecastFiles(b *testing.B) {
	d := openBenchDB(b)
	defer d.Close()
	ctx := logg.WithDiscardHandler(context.Background())
	blob := makeBlob()

	// Each iteration's seed must occupy a disjoint reference_time range to
	// avoid colliding with the forecasts.UNIQUE(reference_time) constraint.
	// One iteration spans 10 forecasts * 3 h = 30 h; pad to a comfortable
	// multi-year gap so the cutoff math stays trivial.
	const iterStride = 10 * 365 * 24 * time.Hour

	expectedDeletions := int64(realisticNumForecasts * totalForecastFiles())
	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		b.StopTimer()
		iterBase := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).
			Add(time.Duration(i) * iterStride)
		for j := range realisticNumForecasts {
			seedRealisticForecast(b, d, iterBase.Add(time.Duration(j)*realisticForecastSpacing), blob)
		}
		// Cutoff well after every seeded valid_time so all 1650 rows are stale.
		cutoff := iterBase.Add(iterStride / 2)
		b.StartTimer()

		n, err := d.QueryRW().DeleteOutdatedForecastFiles(ctx, cutoff)
		if err != nil {
			b.Fatal(err)
		}
		if n != expectedDeletions {
			b.Fatalf("expected %d deletions, got %d", expectedDeletions, n)
		}
	}
}
