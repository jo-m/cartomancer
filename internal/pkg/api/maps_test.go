package api_test

import (
	"database/sql"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
)

func insertMapBuild(t *testing.T, env *testEnv, ready bool) db.MapBuild {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	params := db.InsertMapBuildParams{
		Uuid:      id.String(),
		CreatedAt: now,
		Key:       "20260424.pmtiles",
		Size:      1024,
		Md5sum:    "abc123",
		Uploaded:  now,
		Version:   "3",
		Maxzoom:   14,
	}
	require.NoError(t, env.d.QueryRW().InsertMapBuild(t.Context(), params))
	if ready {
		_, err = env.d.QueryRW().SetMapBuildReady(t.Context(), id.String())
		require.NoError(t, err)
	}
	build, err := env.d.QueryRO().GetReadyMapBuildByUUID(t.Context(), id.String())
	if !ready {
		return db.MapBuild{Uuid: id.String()}
	}
	require.NoError(t, err)
	return build
}

func writeMapFile(t *testing.T, env *testEnv, uuid string, content []byte) {
	t.Helper()
	path := filepath.Join(env.mapsDir, uuid+".pmtiles")
	require.NoError(t, os.WriteFile(path, content, 0600))
}

func TestListMapBuilds_Empty(t *testing.T) {
	env := newTestEnv(t)
	client := env.newClient()

	var resp []map[string]any
	status, _ := env.do(client, http.MethodGet, "/maps", nil, &resp)
	require.Equal(t, http.StatusOK, status)
	require.Empty(t, resp)
}

func TestListMapBuilds_OnlyReady(t *testing.T) {
	env := newTestEnv(t)
	client := env.newClient()

	readyBuild := insertMapBuild(t, env, true)
	insertMapBuild(t, env, false)

	var resp []map[string]any
	status, _ := env.do(client, http.MethodGet, "/maps", nil, &resp)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, resp, 1)
	require.Equal(t, readyBuild.Uuid, resp[0]["uuid"])
	require.Equal(t, float64(14), resp[0]["maxZoom"])
}

func TestGetMapFile_NotFound(t *testing.T) {
	env := newTestEnv(t)
	client := env.newClient()

	status, _ := env.do(client, http.MethodGet, "/maps/"+uuid.NewString(), nil, nil)
	require.Equal(t, http.StatusNotFound, status)
}

func TestGetMapFile_NotReady(t *testing.T) {
	env := newTestEnv(t)
	client := env.newClient()

	build := insertMapBuild(t, env, false)
	status, _ := env.do(client, http.MethodGet, "/maps/"+build.Uuid, nil, nil)
	require.Equal(t, http.StatusNotFound, status)
}

func TestGetMapFile_FileNotOnDisk(t *testing.T) {
	env := newTestEnv(t)
	client := env.newClient()

	build := insertMapBuild(t, env, true)
	status, _ := env.do(client, http.MethodGet, "/maps/"+build.Uuid, nil, nil)
	require.Equal(t, http.StatusNotFound, status)
}

func TestGetMapFile_Success(t *testing.T) {
	env := newTestEnv(t)
	client := env.newClient()

	build := insertMapBuild(t, env, true)
	content := []byte("fake pmtiles content for testing")
	writeMapFile(t, env, build.Uuid, content)

	status, body := env.do(client, http.MethodGet, "/maps/"+build.Uuid, nil, nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, content, body)
}

func TestGetMapFile_RangeRequest(t *testing.T) {
	env := newTestEnv(t)

	build := insertMapBuild(t, env, true)
	content := []byte("0123456789abcdefghij")
	writeMapFile(t, env, build.Uuid, content)

	req, err := http.NewRequest(http.MethodGet, env.ts.URL+"/maps/"+build.Uuid, nil)
	require.NoError(t, err)
	req.Header.Set("Range", "bytes=5-9")

	resp, err := env.ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusPartialContent, resp.StatusCode)

	buf, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("56789"), buf)
}

func TestGetMapFile_BboxFields(t *testing.T) {
	env := newTestEnv(t)
	client := env.newClient()

	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	params := db.InsertMapBuildParams{
		Uuid:       id.String(),
		CreatedAt:  now,
		Key:        "20260424.pmtiles",
		Size:       512,
		Md5sum:     "def456",
		Uploaded:   now,
		Version:    "3",
		Maxzoom:    7,
		BboxMinLon: sql.NullFloat64{Valid: true, Float64: 5.96},
		BboxMinLat: sql.NullFloat64{Valid: true, Float64: 45.82},
		BboxMaxLon: sql.NullFloat64{Valid: true, Float64: 10.49},
		BboxMaxLat: sql.NullFloat64{Valid: true, Float64: 47.81},
	}
	require.NoError(t, env.d.QueryRW().InsertMapBuild(t.Context(), params))
	_, err = env.d.QueryRW().SetMapBuildReady(t.Context(), id.String())
	require.NoError(t, err)

	var resp []map[string]any
	status, _ := env.do(client, http.MethodGet, "/maps", nil, &resp)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, resp, 1)
	require.InDelta(t, 5.96, resp[0]["bboxMinLon"], 0.001)
	require.InDelta(t, 45.82, resp[0]["bboxMinLat"], 0.001)
	require.InDelta(t, 10.49, resp[0]["bboxMaxLon"], 0.001)
	require.InDelta(t, 47.81, resp[0]["bboxMaxLat"], 0.001)
}
