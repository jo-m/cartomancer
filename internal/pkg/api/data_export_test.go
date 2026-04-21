package api_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readZipFile reads the named entry from a zip reader.
func readZipFile(t *testing.T, zr *zip.Reader, name string) []byte {
	t.Helper()
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			require.NoError(t, err)
			defer rc.Close()
			data, err := io.ReadAll(rc)
			require.NoError(t, err)
			return data
		}
	}
	t.Fatalf("zip entry %q not found", name)
	return nil
}

// zipFileNames returns the names of all entries in the zip.
func zipFileNames(zr *zip.Reader) []string {
	names := make([]string, len(zr.File))
	for i, f := range zr.File {
		names[i] = f.Name
	}
	return names
}

// doExport performs GET /account/export and returns the parsed zip reader.
func doExport(t *testing.T, e *testEnv, client *http.Client) *zip.Reader {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, e.ts.URL+"/account/export", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/zip", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "cartomancer-export-")

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	return zr
}

func TestExportData_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodGet, "/account/export", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestExportData_EmptyAccount(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	zr := doExport(t, e, client)

	// user.json, tracks.json, track_groups.json.
	names := zipFileNames(zr)
	assert.Contains(t, names, "user.json")
	assert.Contains(t, names, "tracks.json")
	assert.Contains(t, names, "track_groups.json")
	assert.Len(t, names, 3)

	var user map[string]any
	require.NoError(t, json.Unmarshal(readZipFile(t, zr, "user.json"), &user))
	assert.Equal(t, "Alice", user["name"])
	assert.Equal(t, "alice@example.com", user["email"])

	var tracks []any
	require.NoError(t, json.Unmarshal(readZipFile(t, zr, "tracks.json"), &tracks))
	assert.Empty(t, tracks)

	var groups []any
	require.NoError(t, json.Unmarshal(readZipFile(t, zr, "track_groups.json"), &groups))
	assert.Empty(t, groups)
}

func TestExportData_WithTracks(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("bob@example.com", "Bob", "secret11", false)
	client := e.newClient()
	e.login(client, "bob@example.com", "secret11")

	status, result := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := result["uuid"].(string)
	trackName := result["name"].(string)

	zr := doExport(t, e, client)

	// user.json, tracks.json, track_groups.json, + 1 track file.
	names := zipFileNames(zr)
	assert.Len(t, names, 4)
	assert.Contains(t, names, "tracks/"+trackUUID+".gpx")

	var tracks []map[string]any
	require.NoError(t, json.Unmarshal(readZipFile(t, zr, "tracks.json"), &tracks))
	require.Len(t, tracks, 1)
	assert.Equal(t, trackName, tracks[0]["name"])
	assert.Equal(t, trackUUID, tracks[0]["uuid"])
	assert.Equal(t, "tracks/"+trackUUID+".gpx", tracks[0]["filename"])

	// Tags should be a JSON list (even if empty).
	tags, ok := tracks[0]["tags"].([]any)
	require.True(t, ok, "tags should be a JSON array")
	assert.Empty(t, tags)

	// The track file content should match the original.
	origContent, err := os.ReadFile(testGPXFile)
	require.NoError(t, err)
	assert.Equal(t, origContent, readZipFile(t, zr, "tracks/"+trackUUID+".gpx"))
}

func TestExportData_OnlyOwnTracks(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	e.createUser("bob@example.com", "Bob", "secret22", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret11")

	status, _ := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret22")

	zr := doExport(t, e, bob)

	// Only the 3 JSON files, no track files.
	assert.Len(t, zipFileNames(zr), 3)

	var tracks []any
	require.NoError(t, json.Unmarshal(readZipFile(t, zr, "tracks.json"), &tracks))
	assert.Empty(t, tracks)
}
