package meteo

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/geoadmin"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
	"jo-m.ch/go/cartomancer/internal/pkg/meteo/vars"
)

func TestParseISO8601Duration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"P0DT00H00M00S", 0, false},
		{"P0DT10H00M00S", 10 * time.Hour, false},
		{"P1DT00H00M00S", 24 * time.Hour, false},
		{"P1DT12H30M00S", 36*time.Hour + 30*time.Minute, false},
		{"P0DT00H00M45S", 45 * time.Second, false},
		{"PT1H", time.Hour, false},
		{"", 0, true},
		{"P1Y", 0, true},
		{"not-a-duration", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := geoadmin.ParseISO8601Duration(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetLatestForecastReferenceTime_empty(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)
	_, err := d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestCreateForecastFile_and_GetLatest(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)
	validTime := refTime.Add(10 * time.Hour)

	forecast, err := d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		BoundsMinLat:       sql.NullFloat64{Float64: 45.7, Valid: true},
		BoundsMinLon:       sql.NullFloat64{Float64: 5.9, Valid: true},
		BoundsMaxLat:       sql.NullFloat64{Float64: 47.8, Valid: true},
		BoundsMaxLon:       sql.NullFloat64{Float64: 10.5, Valid: true},
		HorizontalGridFile: []byte("grid data"),
		VerticalGridFile:   []byte("vert grid data"),
	})
	require.NoError(t, err)

	f, err := d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
		ValidTime:      validTime,
		ValidUntilTime: validTime.Add(time.Hour),
		Variable:       vars.VarTotPr.Name,
		File:           []byte("grib data"),
		ForecastID:     forecast.ID,
	})
	require.NoError(t, err)
	require.Greater(t, f.ID, int64(0))
	require.Equal(t, vars.VarTotPr.Name, f.Variable)

	latest, err := d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.NoError(t, err)
	require.Equal(t, refTime.UTC(), latest.UTC())
}

func TestCreateForecastFile_uniqueConstraint(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)

	forecast, err := d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		HorizontalGridFile: []byte("grid data"),
		VerticalGridFile:   []byte("vert grid data"),
	})
	require.NoError(t, err)

	insert := func() error {
		_, err := d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
			ValidTime:      refTime.Add(time.Hour),
			ValidUntilTime: refTime.Add(2 * time.Hour),
			Variable:       vars.VarU10m.Name,
			File:           []byte("grib data"),
			ForecastID:     forecast.ID,
		})
		return err
	}

	require.NoError(t, insert())
	require.Error(t, insert(), "duplicate (forecast_id, variable, valid_time) must be rejected")
}

func TestCountDistinctForecastVariables(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)

	// No forecast row: returns 0.
	count, err := d.QueryRO().CountDistinctForecastVariables(ctx, refTime)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	// Create forecast with files for two variables.
	fc, err := d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		HorizontalGridFile: []byte("grid"),
		VerticalGridFile:   []byte("vert"),
	})
	require.NoError(t, err)

	for _, v := range []string{vars.VarU10m.Name, vars.VarV10m.Name} {
		_, err = d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
			ValidTime:      refTime.Add(time.Hour),
			ValidUntilTime: refTime.Add(2 * time.Hour),
			Variable:       v,
			File:           []byte("grib"),
			ForecastID:     fc.ID,
		})
		require.NoError(t, err)
	}

	count, err = d.QueryRO().CountDistinctForecastVariables(ctx, refTime)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
}

func TestIsForecastComplete(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	dl := NewDownloader(d)
	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)

	fc, err := d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		HorizontalGridFile: []byte("grid"),
		VerticalGridFile:   []byte("vert"),
	})
	require.NoError(t, err)

	// Only two of four expected variables.
	for _, v := range []string{vars.VarU10m.Name, vars.VarV10m.Name} {
		_, err = d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
			ValidTime:      refTime.Add(time.Hour),
			ValidUntilTime: refTime.Add(2 * time.Hour),
			Variable:       v,
			File:           []byte("grib"),
			ForecastID:     fc.ID,
		})
		require.NoError(t, err)
	}

	complete, err := dl.isForecastComplete(ctx, refTime)
	require.NoError(t, err)
	require.False(t, complete, "forecast with 2/4 variables should be incomplete")

	// Add remaining variables.
	for _, v := range []string{vars.VarTotPr.Name, vars.VarT2m.Name} {
		_, err = d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
			ValidTime:      refTime.Add(time.Hour),
			ValidUntilTime: refTime.Add(2 * time.Hour),
			Variable:       v,
			File:           []byte("grib"),
			ForecastID:     fc.ID,
		})
		require.NoError(t, err)
	}

	complete, err = dl.isForecastComplete(ctx, refTime)
	require.NoError(t, err)
	require.True(t, complete, "forecast with 4/4 variables should be complete")
}

func TestListForecastFileKeys(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)
	fc, err := d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		HorizontalGridFile: []byte("grid"),
		VerticalGridFile:   []byte("vert"),
	})
	require.NoError(t, err)

	vt1 := refTime.Add(time.Hour)
	vt2 := refTime.Add(2 * time.Hour)
	for _, v := range []string{vars.VarU10m.Name, vars.VarV10m.Name} {
		for _, vt := range []time.Time{vt1, vt2} {
			_, err = d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
				ValidTime:      vt,
				ValidUntilTime: vt.Add(time.Hour),
				Variable:       v,
				File:           []byte("grib"),
				ForecastID:     fc.ID,
			})
			require.NoError(t, err)
		}
	}

	keys, err := d.QueryRO().ListForecastFileKeys(ctx, fc.ID)
	require.NoError(t, err)
	require.Len(t, keys, 4)
}

func TestForecastExistsForReferenceTime(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)

	// Should not exist yet.
	exists, err := d.QueryRO().ForecastExistsForReferenceTime(ctx, refTime)
	require.NoError(t, err)
	require.Equal(t, int64(0), exists)

	// Create the forecast row.
	_, err = d.QueryRW().CreateForecast(ctx, db.CreateForecastParams{
		CreatedAt:          time.Now(),
		ReferenceTime:      refTime,
		HorizontalGridFile: []byte("grid data"),
		VerticalGridFile:   []byte("vert grid data"),
	})
	require.NoError(t, err)

	// Should exist now.
	exists, err = d.QueryRO().ForecastExistsForReferenceTime(ctx, refTime)
	require.NoError(t, err)
	require.Equal(t, int64(1), exists)
}
