package forecast_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db/forecastdb"
	"jo-m.ch/go/cartomancer/internal/pkg/forecast"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

func TestLoad_MultipleLocations(t *testing.T) {
	d := forecastdb.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := seedDB(t, d)

	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}
	lats := []float64{46.9480, 47.3769, 46.2044}
	lons := []float64{7.4474, 8.5417, 6.1432}

	h, err := forecast.Load(ctx, d, refTime, refTime, bbox, lats, lons)
	require.NoError(t, err)
	require.NotNil(t, h)

	// All three Swiss cities should have valid T_2M values.
	for i := range lats {
		v := h.Sample("T_2M", refTime, i)
		require.False(t, math.IsNaN(float64(v)),
			"expected a value at location %d (%.4f, %.4f), got NaN", i, lats[i], lons[i])
		require.Greater(t, v, float32(200))
		require.Less(t, v, float32(340))
	}
}

func TestLoad_OutOfRangeIndex(t *testing.T) {
	d := forecastdb.GetTestDB(t)
	defer d.Close()
	ctx := logg.WithTestLogger(t.Context(), t)

	refTime := seedDB(t, d)

	bbox := forecast.BBox{MinLat: 45, MaxLat: 48, MinLon: 6, MaxLon: 10}
	h, err := forecast.Load(ctx, d, refTime, refTime, bbox, bernLat, bernLon)
	require.NoError(t, err)

	v := h.Sample("T_2M", refTime, 99)
	require.True(t, math.IsNaN(float64(v)))
}
