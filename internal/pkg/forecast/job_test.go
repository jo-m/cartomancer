package forecast

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/blob"
	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
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
			got, err := parseISO8601Duration(tc.input)
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

	ctx := logg.WithDiscardHandler(t.Context())
	_, err := d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestCreateForecastFile_and_GetLatest(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithDiscardHandler(t.Context())

	b, err := blob.Create(ctx, d.QueryRW(), []byte("grib data"), blob.CompressionZstd)
	require.NoError(t, err)

	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)
	validTime := refTime.Add(10 * time.Hour)

	f, err := d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
		CreatedAt:     time.Now(),
		ReferenceTime: refTime,
		ValidTime:     validTime,
		Variable:      "TOT_PREC",
		BoundsMinLat:  sql.NullFloat64{Float64: 45.7, Valid: true},
		BoundsMinLon:  sql.NullFloat64{Float64: 5.9, Valid: true},
		BoundsMaxLat:  sql.NullFloat64{Float64: 47.8, Valid: true},
		BoundsMaxLon:  sql.NullFloat64{Float64: 10.5, Valid: true},
		BlobID:        b.ID,
	})
	require.NoError(t, err)
	require.Greater(t, f.ID, int64(0))
	require.Equal(t, "TOT_PREC", f.Variable)
	require.InDelta(t, 45.7, f.BoundsMinLat.Float64, 1e-9)

	latest, err := d.QueryRO().GetLatestForecastReferenceTime(ctx)
	require.NoError(t, err)
	require.Equal(t, refTime.UTC(), latest.UTC())
}

func TestCreateForecastFile_uniqueConstraint(t *testing.T) {
	d := db.GetTestDB(t)
	defer d.Close()

	ctx := logg.WithDiscardHandler(t.Context())

	insertBlob := func() int64 {
		b, err := blob.Create(ctx, d.QueryRW(), []byte("grib data"), blob.CompressionNone)
		require.NoError(t, err)
		return b.ID
	}

	refTime := time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC)

	insert := func(blobID int64) error {
		_, err := d.QueryRW().CreateForecastFile(ctx, db.CreateForecastFileParams{
			CreatedAt:     time.Now(),
			ReferenceTime: refTime,
			ValidTime:     refTime.Add(time.Hour),
			Variable:      "U_10M",
			BlobID:        blobID,
		})
		return err
	}

	require.NoError(t, insert(insertBlob()))
	require.Error(t, insert(insertBlob()), "duplicate (reference_time, variable, valid_time) must be rejected")
}
