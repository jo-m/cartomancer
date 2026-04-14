package forecast_test

import (
	"context"
	"database/sql"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db/forecastdb"
	"jo-m.ch/go/cartomancer/internal/pkg/forecast"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

const (
	// ~90 km straight line from Bern to Zurich, 200 points.
	benchNumPoints = 200
	benchStartLat  = 46.948479 // Bern
	benchStartLon  = 7.447443
	benchEndLat    = 47.375189 // Zurich
	benchEndLon    = 8.539299
)

// benchLine returns 200 evenly spaced points on a ~90 km straight line from Bern to Zurich.
func benchLine() (lats, lons []float64) {
	lats = make([]float64, benchNumPoints)
	lons = make([]float64, benchNumPoints)
	for i := range benchNumPoints {
		f := float64(i) / float64(benchNumPoints-1)
		lats[i] = benchStartLat + f*(benchEndLat-benchStartLat)
		lons[i] = benchStartLon + f*(benchEndLon-benchStartLon)
	}
	return lats, lons
}

// seedBenchDB inserts the grid constants and multiple forecast files (T_2M and
// TOT_PREC at 5 hourly time steps each) to simulate a realistic workload.
func seedBenchDB(b *testing.B, d *forecastdb.DB) time.Time {
	b.Helper()
	ctx := logg.WithDiscardHandler(context.Background())
	refTime := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	gridContent, err := os.ReadFile(gridTestdata)
	require.NoError(b, err)

	fc, err := d.QueryRW().CreateForecast(ctx, forecastdb.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		BoundsMinLat:       sql.NullFloat64{Float64: 43.0, Valid: true},
		BoundsMinLon:       sql.NullFloat64{Float64: 2.0, Valid: true},
		BoundsMaxLat:       sql.NullFloat64{Float64: 50.0, Valid: true},
		BoundsMaxLon:       sql.NullFloat64{Float64: 16.0, Valid: true},
		HorizontalGridFile: gridContent,
		VerticalGridFile:   []byte("vert"),
	})
	require.NoError(b, err)

	t2mContent, err := os.ReadFile(t2mTestdata)
	require.NoError(b, err)
	totPrContent, err := os.ReadFile("../grib2/testdata/tot_pr_0h.grib2")
	require.NoError(b, err)

	// Insert 5 hourly time steps for each variable.
	for h := range 5 {
		vt := refTime.Add(time.Duration(h) * time.Hour)
		_, err = d.QueryRW().CreateForecastFile(ctx, forecastdb.CreateForecastFileParams{
			ValidTime:      vt,
			ValidUntilTime: vt.Add(time.Hour),
			Variable:       "T_2M",
			File:           t2mContent,
			ForecastID:     fc.ID,
		})
		require.NoError(b, err)

		_, err = d.QueryRW().CreateForecastFile(ctx, forecastdb.CreateForecastFileParams{
			ValidTime:      vt,
			ValidUntilTime: vt.Add(time.Hour),
			Variable:       "TOT_PREC",
			File:           totPrContent,
			ForecastID:     fc.ID,
		})
		require.NoError(b, err)
	}

	return refTime
}

// openBenchDB creates a temporary forecast database for benchmarks.
func openBenchDB(b *testing.B) *forecastdb.DB {
	b.Helper()
	dir := b.TempDir()
	ctx := logg.WithDiscardHandler(context.Background())
	d, err := forecastdb.Open(ctx, filepath.Join(dir, "forecast.db"))
	require.NoError(b, err)
	return d
}

// heapInUse forces a GC and returns the current heap in-use in bytes.
func heapInUse() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapInuse
}

// BenchmarkLoad benchmarks loading forecast data and sampling 200 points
// along the ~90 km Bern-Zurich line (2 variables, 5 hourly time steps).
func BenchmarkLoad(b *testing.B) {
	d := openBenchDB(b)
	defer d.Close()
	ctx := logg.WithDiscardHandler(context.Background())

	refTime := seedBenchDB(b, d)
	endTime := refTime.Add(5 * time.Hour)
	bbox := forecast.BBox{MinLat: 46.5, MaxLat: 47.5, MinLon: 7.0, MaxLon: 9.0}
	lats, lons := benchLine()

	var maxHeap uint64

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		h, err := forecast.Load(ctx, d, refTime, endTime, bbox, lats, lons)
		if err != nil {
			b.Fatal(err)
		}

		if heap := heapInUse(); heap > maxHeap {
			maxHeap = heap
		}

		for i := range benchNumPoints {
			pointTime := refTime.Add(time.Duration(float64(i)/float64(benchNumPoints)*4.9) * time.Hour)
			v := h.Sample("T_2M", pointTime, i)
			if math.IsNaN(float64(v)) {
				b.Fatal("unexpected NaN for T_2M")
			}
			h.Sample("TOT_PREC", pointTime, i)
		}
	}
	b.ReportMetric(float64(maxHeap)/1024/1024, "peak-heap-MB")
}
