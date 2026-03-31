package forecast_test

import (
	"database/sql"
	"math"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/forecast"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

const (
	gridTestdata = "../grib2/testdata/horiz_const.grib2"
	t2mTestdata  = "../grib2/testdata/t_2m_0h.grib2"
)

// seedDB inserts the grid constants and a forecast file into the test database.
// Returns the reference time used.
func seedDB(t *testing.T, d *db.DB) time.Time {
	t.Helper()
	ctx := logg.WithTestLogger(t.Context(), t)
	refTime := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	// Insert forecast row with grid constants and bounds.
	gridContent, err := os.ReadFile(gridTestdata)
	require.NoError(t, err)

	fc, err := d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		BoundsMinLat:       sql.NullFloat64{Float64: 43.0, Valid: true},
		BoundsMinLon:       sql.NullFloat64{Float64: 2.0, Valid: true},
		BoundsMaxLat:       sql.NullFloat64{Float64: 50.0, Valid: true},
		BoundsMaxLon:       sql.NullFloat64{Float64: 16.0, Valid: true},
		HorizontalGridFile: gridContent,
		VerticalGridFile:   []byte("vert grid"),
	})
	require.NoError(t, err)

	// Insert a T_2M forecast file at 0h horizon (valid_time == reference_time).
	t2mContent, err := os.ReadFile(t2mTestdata)
	require.NoError(t, err)

	_, err = d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
		ValidTime:      refTime,
		ValidUntilTime: refTime.Add(time.Hour),
		Variable:       "T_2M",
		File:           t2mContent,
		ForecastID:     fc.ID,
	})
	require.NoError(t, err)

	return refTime
}

func TestLoad_NoData(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}

	h, err := forecast.Load(ctx, d, start, end, bbox)
	require.ErrorIs(t, err, forecast.ErrNoData)
	require.Nil(t, h)
}

func TestLoad_FullCoverage(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := seedDB(t, d)

	// Query exactly the time of the single file.
	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}
	h, err := forecast.Load(ctx, d, refTime, refTime, bbox)
	require.NoError(t, err)
	require.NotNil(t, h)
	require.Contains(t, h.Variables(), "T_2M")
}

func TestLoad_Incomplete(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := seedDB(t, d)

	// Query a wider window that extends beyond the single file's valid time.
	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}
	h, err := forecast.Load(ctx, d, refTime, refTime.Add(6*time.Hour), bbox)
	require.ErrorIs(t, err, forecast.ErrIncomplete)
	require.NotNil(t, h)
	require.Contains(t, h.Variables(), "T_2M")
}

func TestSample_KnownCity(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := seedDB(t, d)

	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}
	h, err := forecast.Load(ctx, d, refTime, refTime, bbox)
	require.NoError(t, err)

	// T_2M at Bern in Kelvin: expect a reasonable temperature.
	v := h.Sample("T_2M", refTime, 46.9480, 7.4474)
	require.False(t, math.IsNaN(float64(v)), "expected a value, got NaN")
	require.Greater(t, v, float32(200))
	require.Less(t, v, float32(340))
}

func TestSample_UnknownVariable(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := seedDB(t, d)

	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}
	h, err := forecast.Load(ctx, d, refTime, refTime, bbox)
	require.NoError(t, err)

	v := h.Sample("NONEXISTENT", refTime, 46.9480, 7.4474)
	require.True(t, math.IsNaN(float64(v)))
}

func TestSample_OutsideDomain(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := seedDB(t, d)

	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}
	h, err := forecast.Load(ctx, d, refTime, refTime, bbox)
	require.NoError(t, err)

	// Far outside the ICON-CH1 domain.
	v := h.Sample("T_2M", refTime, 0, 0)
	require.True(t, math.IsNaN(float64(v)))
}

func TestLoad_FallbackToOlderForecast(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	gridContent, err := os.ReadFile(gridTestdata)
	require.NoError(t, err)
	t2mContent, err := os.ReadFile(t2mTestdata)
	require.NoError(t, err)

	refTimeOld := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	refTimeNew := time.Date(2026, 3, 10, 6, 0, 0, 0, time.UTC)

	// Create an older forecast with T_2M at 0h and 1h.
	fcOld, err := d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTimeOld,
		BoundsMinLat:       sql.NullFloat64{Float64: 43.0, Valid: true},
		BoundsMinLon:       sql.NullFloat64{Float64: 2.0, Valid: true},
		BoundsMaxLat:       sql.NullFloat64{Float64: 50.0, Valid: true},
		BoundsMaxLon:       sql.NullFloat64{Float64: 16.0, Valid: true},
		HorizontalGridFile: gridContent,
		VerticalGridFile:   []byte("vert"),
	})
	require.NoError(t, err)
	for _, h := range []int{0, 1} {
		vt := refTimeOld.Add(time.Duration(h) * time.Hour)
		_, err = d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
			ValidTime:      vt,
			ValidUntilTime: vt.Add(time.Hour),
			Variable:       "T_2M",
			File:           t2mContent,
			ForecastID:     fcOld.ID,
		})
		require.NoError(t, err)
	}

	// Create a newer forecast with T_2M only at 1h (missing 0h).
	fcNew, err := d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTimeNew,
		BoundsMinLat:       sql.NullFloat64{Float64: 43.0, Valid: true},
		BoundsMinLon:       sql.NullFloat64{Float64: 2.0, Valid: true},
		BoundsMaxLat:       sql.NullFloat64{Float64: 50.0, Valid: true},
		BoundsMaxLon:       sql.NullFloat64{Float64: 16.0, Valid: true},
		HorizontalGridFile: gridContent,
		VerticalGridFile:   []byte("vert"),
	})
	require.NoError(t, err)
	vt := refTimeOld.Add(time.Hour) // valid_time = 01:00, same as old forecast's 1h file
	_, err = d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
		ValidTime:      vt,
		ValidUntilTime: vt.Add(time.Hour),
		Variable:       "T_2M",
		File:           t2mContent,
		ForecastID:     fcNew.ID,
	})
	require.NoError(t, err)

	// Query for [0h, 2h) -- old forecast covers 0h, new covers 1h.
	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}
	h, err := forecast.Load(ctx, d, refTimeOld, refTimeOld.Add(2*time.Hour), bbox)
	require.NoError(t, err)
	require.NotNil(t, h)

	// Both time slots should have data (0h from old, 1h from new).
	v0 := h.Sample("T_2M", refTimeOld, 46.9480, 7.4474)
	require.False(t, isNaN(v0), "0h slot should be filled by older forecast")
	v1 := h.Sample("T_2M", refTimeOld.Add(time.Hour), 46.9480, 7.4474)
	require.False(t, isNaN(v1), "1h slot should be filled by newer forecast")
}

func isNaN(v float32) bool {
	return math.IsNaN(float64(v))
}

func TestSample_BeforeFirstMessage(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := seedDB(t, d)

	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}
	h, err := forecast.Load(ctx, d, refTime, refTime, bbox)
	require.NoError(t, err)

	// One hour before the reference time: no message covers this.
	v := h.Sample("T_2M", refTime.Add(-time.Hour), 46.9480, 7.4474)
	require.True(t, math.IsNaN(float64(v)))
}
