package rest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/db"
)

func TestAdminListForecasts_Empty(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	var resp map[string]any
	status, _ := e.do(client, http.MethodGet, "/admin/forecasts", nil, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Len(t, resp["forecasts"], 0)
}

func TestAdminListForecasts_WithData(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("admin@example.com", "Admin", "adminpass", true)
	client := e.newClient()
	e.login(client, "admin@example.com", "adminpass")

	now := time.Now().UTC().Truncate(time.Second)
	fc, err := e.d.QueryRW().CreateForecast(t.Context(), db.CreateForecastParams{
		CreatedAt:          now,
		ReferenceTime:      now,
		HorizontalGridFile: []byte("hgrid"),
		VerticalGridFile:   []byte("vgrid"),
		Attribution:        "test-source",
		AttributionHref:    "https://example.com",
	})
	require.NoError(t, err)

	_, err = e.d.QueryRW().CreateForecastFile(t.Context(), db.CreateForecastFileParams{
		ValidTime:      now,
		ValidUntilTime: now.Add(time.Hour),
		Variable:       "U_10M",
		File:           []byte("grib-data-here"),
		ForecastID:     fc.ID,
	})
	require.NoError(t, err)

	var resp map[string]any
	status, _ := e.do(client, http.MethodGet, "/admin/forecasts", nil, &resp)
	assert.Equal(t, http.StatusOK, status)

	forecasts := resp["forecasts"].([]any)
	require.Len(t, forecasts, 1)

	forecast := forecasts[0].(map[string]any)
	assert.Equal(t, "test-source", forecast["attribution"])
	assert.Equal(t, "https://example.com", forecast["attributionHref"])

	files := forecast["files"].([]any)
	require.Len(t, files, 1)

	file := files[0].(map[string]any)
	assert.Equal(t, "U_10M", file["variable"])
	assert.Equal(t, float64(len("grib-data-here")), file["fileSize"])
}

func TestAdminListForecasts_Forbidden(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodGet, "/admin/forecasts", nil, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestAdminListForecasts_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodGet, "/admin/forecasts", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}
