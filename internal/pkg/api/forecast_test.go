package api_test

import (
	"database/sql"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/db/forecastdb"
)

const (
	gridTestdata = "../grib2/testdata/horiz_const.grib2"
	t2mTestdata  = "../grib2/testdata/t_2m_0h.grib2"
)

// seedForecastDB inserts forecast data into the test database.
func seedForecastDB(t *testing.T, d *forecastdb.DB, refTime time.Time) {
	t.Helper()
	ctx := t.Context()

	gridContent, err := os.ReadFile(gridTestdata)
	require.NoError(t, err)

	fc, err := d.QueryRW().CreateForecast(ctx, forecastdb.CreateForecastParams{
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

	t2mContent, err := os.ReadFile(t2mTestdata)
	require.NoError(t, err)

	_, err = d.QueryRW().CreateForecastFile(ctx, forecastdb.CreateForecastFileParams{
		ValidTime:      refTime,
		ValidUntilTime: refTime.Add(time.Hour),
		Variable:       "T_2M",
		File:           t2mContent,
		ForecastID:     fc.ID,
	})
	require.NoError(t, err)
}

func TestGetTrackForecast_Unauthenticated_NotFound(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodGet, "/tracks/nonexistent/forecast?startTime=2026-03-10T00:00:00Z&speedKmh=25", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackForecast_Unauthenticated_PrivateTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	owner := e.newClient()
	e.login(owner, "alice@example.com", "secret11")

	status, resp := e.doUpload(owner, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid := resp["uuid"].(string)

	// Anonymous client cannot see a private track.
	anon := e.newClient()
	status, _ = e.do(anon, http.MethodGet, "/tracks/"+uuid+"/forecast?startTime=2026-03-10T00:00:00Z&speedKmh=25", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackForecast_NotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodGet, "/tracks/nonexistent/forecast?startTime=2026-03-10T00:00:00Z&speedKmh=25", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackForecast_MissingParams(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, resp := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid := resp["uuid"].(string)

	// Missing startTime.
	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid+"/forecast?speedKmh=25", nil, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	// Missing speedKmh.
	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid+"/forecast?startTime=2026-03-10T00:00:00Z", nil, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestGetTrackForecast_NoForecastData(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, resp := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid := resp["uuid"].(string)

	var result map[string]any
	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid+"/forecast?startTime=2026-03-10T00:00:00Z&speedKmh=25", nil, &result)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "none", result["forecastStatus"])

	points, ok := result["points"].([]any)
	require.True(t, ok)
	require.Greater(t, len(points), 0)

	first := points[0].(map[string]any)
	assert.Equal(t, float64(0), first["distanceM"])
	assert.NotEmpty(t, first["time"])
	assert.Nil(t, first["temperatureC"])
	assert.Nil(t, first["precipitationRate"])
}

func TestGetTrackForecast_PolylineNotBackfilled(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, resp := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid := resp["uuid"].(string)

	// Clear the precomputed polylines to simulate a track that has not yet
	// been backfilled. The forecast endpoint must surface this as a 500.
	require.NoError(t, e.d.QueryRW().SetTrackPreviewPolylines(t.Context(), db.SetTrackPreviewPolylinesParams{
		Uuid:                uuid,
		PolylineDp5mVarint:  []byte{},
		PolylineDp50mVarint: []byte{},
	}))

	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid+"/forecast?startTime=2026-03-10T00:00:00Z&speedKmh=25", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, status)
}

func TestGetTrackForecast_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, resp := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid := resp["uuid"].(string)

	refTime := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	seedForecastDB(t, e.fd, refTime)

	var result map[string]any
	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid+"/forecast?startTime=2026-03-10T00:00:00Z&speedKmh=25", nil, &result)
	assert.Equal(t, http.StatusOK, status)
	// The seeded test data covers one 1h step starting at refTime, which is
	// enough for the short test track at 25 km/h.
	assert.Equal(t, "full", result["forecastStatus"])

	points, ok := result["points"].([]any)
	require.True(t, ok)
	require.Greater(t, len(points), 0)

	first := points[0].(map[string]any)
	assert.Equal(t, float64(0), first["distanceM"])
	assert.NotEmpty(t, first["time"])
	assert.Contains(t, first, "temperatureC")
	assert.Contains(t, first, "precipitationRate")
}
