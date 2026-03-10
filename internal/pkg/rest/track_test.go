package rest_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testGPXFile = "../load/testdata/COURSE_436298480.gpx"
const testGPXFile2 = "../load/testdata/2022-05-25_781839002_BP LA - Granadilla - Vilaflor - Boca Tauce - Arona (Road).gpx"

// doUpload sends a multipart POST /tracks request with the given file.
func (e *testEnv) doUpload(client *http.Client, filename string) (int, map[string]any) {
	e.t.Helper()

	content, err := os.ReadFile(filename)
	require.NoError(e.t, err)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filepath.Base(filename)+`"`)
	h.Set("Content-Type", "application/gpx+xml")
	part, err := mw.CreatePart(h)
	require.NoError(e.t, err)
	_, err = part.Write(content)
	require.NoError(e.t, err)
	require.NoError(e.t, mw.Close())

	req, err := http.NewRequest(http.MethodPost, e.ts.URL+"/tracks", &buf)
	require.NoError(e.t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := client.Do(req)
	require.NoError(e.t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)

	var result map[string]any
	if len(raw) > 0 {
		require.NoError(e.t, json.Unmarshal(raw, &result))
	}

	return resp.StatusCode, result
}

func TestUploadTrack_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.doUpload(client, testGPXFile)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestUploadTrack_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, resp := e.doUpload(client, testGPXFile)
	assert.Equal(t, http.StatusCreated, status)
	assert.NotEmpty(t, resp["uuid"])
	assert.NotEmpty(t, resp["name"])
	assert.Greater(t, resp["totalDistanceM"], float64(0))
}

func TestUploadTrack_AppearsInList(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, resp := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID, ok := resp["uuid"].(string)
	require.True(t, ok)

	var listResp map[string]any
	status, _ = e.do(client, http.MethodGet, "/tracks", nil, &listResp)
	assert.Equal(t, http.StatusOK, status)

	tracks, _ := listResp["tracks"].([]any)
	require.Len(t, tracks, 1)
	track := tracks[0].(map[string]any)
	assert.Equal(t, trackUUID, track["uuid"])
}

func TestUploadTrack_GetByUUID(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, uploaded := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID, ok := uploaded["uuid"].(string)
	require.True(t, ok)

	var track map[string]any
	status, _ = e.do(client, http.MethodGet, "/tracks/"+trackUUID, nil, &track)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, trackUUID, track["uuid"])
	assert.Equal(t, uploaded["name"], track["name"])
}

func TestUploadTrack_OtherUserCannotSee(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")

	status, _ = e.do(bob, http.MethodGet, "/tracks/"+trackUUID, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestUploadTrack_DuplicateRejected(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)

	// Second upload of the same file must be rejected.
	status, _ = e.doUpload(client, testGPXFile)
	assert.Equal(t, http.StatusConflict, status)
}

func TestUploadTrack_DuplicateAllowedForDifferentUser(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	status, _ := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)

	// Same file uploaded by a different user must succeed.
	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")
	status, _ = e.doUpload(bob, testGPXFile)
	assert.Equal(t, http.StatusCreated, status)
}
