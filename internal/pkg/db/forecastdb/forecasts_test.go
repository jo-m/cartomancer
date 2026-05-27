package forecastdb_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db/forecastdb"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

// seedHourlyForecast inserts one forecast row plus hourly files for the given
// variable at lead times [0, steps). Each file's valid_until_time is
// valid_time + 1h, matching the production convention used by meteo/job.go.
func seedHourlyForecast(t *testing.T, d *forecastdb.DB, refTime time.Time, variable string, steps int) int64 {
	t.Helper()
	ctx := logg.WithTestLogger(t.Context(), t)

	fc, err := d.QueryRW().CreateForecast(ctx, forecastdb.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		BoundsMinLat:       sql.NullFloat64{Float64: 43.0, Valid: true},
		BoundsMinLon:       sql.NullFloat64{Float64: 2.0, Valid: true},
		BoundsMaxLat:       sql.NullFloat64{Float64: 50.0, Valid: true},
		BoundsMaxLon:       sql.NullFloat64{Float64: 16.0, Valid: true},
		HorizontalGridFile: []byte("hgrid"),
		VerticalGridFile:   []byte("vgrid"),
	})
	require.NoError(t, err)

	for step := range steps {
		vt := refTime.Add(time.Duration(step) * time.Hour)
		_, err := d.QueryRW().CreateForecastFile(ctx, forecastdb.CreateForecastFileParams{
			ValidTime:      vt,
			ValidUntilTime: vt.Add(time.Hour),
			Variable:       variable,
			File:           []byte("payload"),
			ForecastID:     fc.ID,
		})
		require.NoError(t, err)
	}
	return fc.ID
}

// queryWindow runs ListForecastFileIDsForWindow with a Swiss-domain bbox and
// returns the rows.
func queryWindow(t *testing.T, d *forecastdb.DB, start, end time.Time) []forecastdb.ListForecastFileIDsForWindowRow {
	t.Helper()
	ctx := logg.WithTestLogger(t.Context(), t)
	rows, err := d.QueryRO().ListForecastFileIDsForWindow(ctx, forecastdb.ListForecastFileIDsForWindowParams{
		Start:  start,
		End:    end,
		MaxLat: sql.NullFloat64{Float64: 48.0, Valid: true},
		MinLat: sql.NullFloat64{Float64: 45.0, Valid: true},
		MaxLon: sql.NullFloat64{Float64: 10.0, Valid: true},
		MinLon: sql.NullFloat64{Float64: 6.0, Valid: true},
	})
	require.NoError(t, err)
	return rows
}

// TestListForecastFileIDsForWindow_IncludesPredecessor verifies that the query
// returns the file whose validity ends at or before `start`, so that the
// loader has a same-run predecessor available for de-averaging running-mean
// and running-accumulation variables.
func TestListForecastFileIDsForWindow_IncludesPredecessor(t *testing.T) {
	d := forecastdb.GetTestDB(t)
	defer d.Close()

	refTime := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	seedHourlyForecast(t, d, refTime, "ASWDIR_S", 5)

	// Start partway through the third hourly file's validity (valid_time = +2h,
	// valid_until_time = +3h). The predecessor with valid_time = +1h ends
	// exactly at +2h, before `start`.
	start := refTime.Add(2*time.Hour + 30*time.Minute)
	end := refTime.Add(4*time.Hour + 30*time.Minute)

	rows := queryWindow(t, d, start, end)

	validTimes := make([]time.Time, len(rows))
	for i, r := range rows {
		validTimes[i] = r.ValidTime.UTC()
	}

	require.Equal(t, []time.Time{
		refTime.Add(1 * time.Hour),
		refTime.Add(2 * time.Hour),
		refTime.Add(3 * time.Hour),
		refTime.Add(4 * time.Hour),
	}, validTimes, "predecessor at +1h must be included alongside the in-window files")
}

// TestListForecastFileIDsForWindow_NoPredecessorAvailable verifies that the
// query still works when no file exists before `start`: the in-window files
// alone are returned, no predecessor row is invented.
func TestListForecastFileIDsForWindow_NoPredecessorAvailable(t *testing.T) {
	d := forecastdb.GetTestDB(t)
	defer d.Close()

	refTime := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	seedHourlyForecast(t, d, refTime, "ASWDIR_S", 3)

	// start before the first file; the first file itself satisfies the
	// valid_until_time > start condition, so no predecessor lookup is needed
	// and no row should appear at a time earlier than refTime.
	start := refTime.Add(-30 * time.Minute)
	end := refTime.Add(3 * time.Hour)

	rows := queryWindow(t, d, start, end)
	require.Len(t, rows, 3)
	require.Equal(t, refTime, rows[0].ValidTime.UTC())
}

// TestListForecastFileIDsForWindow_PredecessorPerVariable verifies that the
// predecessor lookup runs independently per variable: each variable's first
// in-window file gets its own preceding file included if one exists.
func TestListForecastFileIDsForWindow_PredecessorPerVariable(t *testing.T) {
	d := forecastdb.GetTestDB(t)
	defer d.Close()

	refTime := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	ctx := logg.WithTestLogger(t.Context(), t)

	fc, err := d.QueryRW().CreateForecast(ctx, forecastdb.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		BoundsMinLat:       sql.NullFloat64{Float64: 43.0, Valid: true},
		BoundsMinLon:       sql.NullFloat64{Float64: 2.0, Valid: true},
		BoundsMaxLat:       sql.NullFloat64{Float64: 50.0, Valid: true},
		BoundsMaxLon:       sql.NullFloat64{Float64: 16.0, Valid: true},
		HorizontalGridFile: []byte("hgrid"),
		VerticalGridFile:   []byte("vgrid"),
	})
	require.NoError(t, err)

	for _, variable := range []string{"ASWDIR_S", "ASWDIFD_S", "T_2M"} {
		for step := range 4 {
			vt := refTime.Add(time.Duration(step) * time.Hour)
			_, err := d.QueryRW().CreateForecastFile(ctx, forecastdb.CreateForecastFileParams{
				ValidTime:      vt,
				ValidUntilTime: vt.Add(time.Hour),
				Variable:       variable,
				File:           []byte("payload"),
				ForecastID:     fc.ID,
			})
			require.NoError(t, err)
		}
	}

	start := refTime.Add(2*time.Hour + 30*time.Minute)
	end := refTime.Add(2*time.Hour + 45*time.Minute)

	rows := queryWindow(t, d, start, end)

	byVar := map[string][]time.Time{}
	for _, r := range rows {
		byVar[r.Variable] = append(byVar[r.Variable], r.ValidTime.UTC())
	}

	expected := []time.Time{refTime.Add(1 * time.Hour), refTime.Add(2 * time.Hour)}
	for _, variable := range []string{"ASWDIR_S", "ASWDIFD_S", "T_2M"} {
		require.Equal(t, expected, byVar[variable], "variable %s missing expected files", variable)
	}
}
