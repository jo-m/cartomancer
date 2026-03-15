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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testGPXFile = "../load/testdata/COURSE_436298480.gpx"
const testGPXFile2 = "../load/testdata/2022-05-25_781839002_BP LA - Granadilla - Vilaflor - Boca Tauce - Arona (Road).gpx"

// doUpload sends a multipart POST /tracks request with the given file.
// doUploadRaw sends a multipart POST /tracks request with given content and upload filename.
func (e *testEnv) doUploadRaw(client *http.Client, content []byte, uploadFilename string) (int, map[string]any) {
	e.t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+uploadFilename+`"`)
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

func (e *testEnv) doUpload(client *http.Client, filename string) (int, map[string]any) {
	e.t.Helper()

	content, err := os.ReadFile(filename)
	require.NoError(e.t, err)

	return e.doUploadRaw(client, content, filepath.Base(filename))
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

// makeTrackPublic patches the track to set public=true.
func (e *testEnv) makeTrackPublic(client *http.Client, trackUUID string, trackName string) {
	e.t.Helper()
	status, _ := e.do(client, http.MethodPatch, "/tracks/"+trackUUID, map[string]any{
		"name":   trackName,
		"public": true,
		"tags":   []string{},
	}, nil)
	require.Equal(e.t, http.StatusOK, status)
}

func TestListTracks_Unauthenticated_OnlyPublic(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	// Alice uploads two tracks; makes one public.
	status, uploaded1 := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid1 := uploaded1["uuid"].(string)
	status, uploaded2 := e.doUpload(alice, testGPXFile2)
	require.Equal(t, http.StatusCreated, status)
	e.makeTrackPublic(alice, uploaded2["uuid"].(string), uploaded2["name"].(string))

	// Unauthenticated client sees only the public track.
	anon := e.newClient()
	var listResp map[string]any
	status, _ = e.do(anon, http.MethodGet, "/tracks", nil, &listResp)
	assert.Equal(t, http.StatusOK, status)
	tracks, _ := listResp["tracks"].([]any)
	require.Len(t, tracks, 1)
	assert.NotEqual(t, uuid1, tracks[0].(map[string]any)["uuid"])
}

func TestListTracks_OnlyMine(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	status, aliceTrack := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	e.makeTrackPublic(alice, aliceTrack["uuid"].(string), aliceTrack["name"].(string))

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")
	status, _ = e.doUpload(bob, testGPXFile)
	require.Equal(t, http.StatusCreated, status)

	// Bob with onlyMine=true should see only his own track, not Alice's public one.
	var listResp map[string]any
	status, _ = e.do(bob, http.MethodGet, "/tracks?onlyMine=true", nil, &listResp)
	assert.Equal(t, http.StatusOK, status)
	tracks, _ := listResp["tracks"].([]any)
	require.Len(t, tracks, 1)
	assert.Equal(t, "bob@example.com", "bob@example.com") // sanity
	assert.NotEqual(t, aliceTrack["uuid"], tracks[0].(map[string]any)["uuid"])
}

func TestGetTrack_Unauthenticated_PublicTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)
	e.makeTrackPublic(alice, trackUUID, uploaded["name"].(string))

	// Unauthenticated client can fetch a public track.
	anon := e.newClient()
	var track map[string]any
	status, _ = e.do(anon, http.MethodGet, "/tracks/"+trackUUID, nil, &track)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, trackUUID, track["uuid"])
}

func TestGetTrack_Unauthenticated_PrivateTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)

	// Unauthenticated client cannot fetch a private track.
	anon := e.newClient()
	status, _ = e.do(anon, http.MethodGet, "/tracks/"+trackUUID, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackSVG_Unauthenticated_PublicTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)
	e.makeTrackPublic(alice, trackUUID, uploaded["name"].(string))

	// Unauthenticated client can fetch the SVG preview of a public track.
	anon := e.newClient()
	req, err := http.NewRequest(http.MethodGet, e.ts.URL+"/tracks/"+trackUUID+"/preview.svg", nil)
	require.NoError(t, err)
	resp, err := anon.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/svg+xml", resp.Header.Get("Content-Type"))
}

// setTags patches the given tags onto a track.
func (e *testEnv) setTags(client *http.Client, trackUUID string, trackName string, tags []string) {
	e.t.Helper()
	status, _ := e.do(client, http.MethodPatch, "/tracks/"+trackUUID, map[string]any{
		"name": trackName,
		"tags": tags,
	}, nil)
	require.Equal(e.t, http.StatusOK, status)
}

func TestListTracks_FilterBySport(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, track1 := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)

	// sport=1 (Running) should return no results since the test file is cycling.
	var listResp map[string]any
	status, _ = e.do(alice, http.MethodGet, "/tracks?onlyMine=true&sport=1", nil, &listResp)
	assert.Equal(t, http.StatusOK, status)
	tracks, _ := listResp["tracks"].([]any)
	for _, tr := range tracks {
		assert.NotEqual(t, track1["uuid"], tr.(map[string]any)["uuid"])
	}
}

func TestListTracks_FilterByTagOR(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, track1 := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	status, track2 := e.doUpload(alice, testGPXFile2)
	require.Equal(t, http.StatusCreated, status)

	e.setTags(alice, track1["uuid"].(string), track1["name"].(string), []string{"alpine", "race"})
	e.setTags(alice, track2["uuid"].(string), track2["name"].(string), []string{"road"})

	// OR mode: tag=alpine OR tag=road should return both tracks.
	var listResp map[string]any
	status, _ = e.do(alice, http.MethodGet, "/tracks?onlyMine=true&tag=alpine&tag=road", nil, &listResp)
	assert.Equal(t, http.StatusOK, status)
	tracks, _ := listResp["tracks"].([]any)
	assert.Len(t, tracks, 2)

	// Only tag=alpine should return only track1.
	status, _ = e.do(alice, http.MethodGet, "/tracks?onlyMine=true&tag=alpine", nil, &listResp)
	assert.Equal(t, http.StatusOK, status)
	tracks, _ = listResp["tracks"].([]any)
	require.Len(t, tracks, 1)
	assert.Equal(t, track1["uuid"], tracks[0].(map[string]any)["uuid"])
}

func TestListTracks_FilterByTagAND(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, track1 := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	status, track2 := e.doUpload(alice, testGPXFile2)
	require.Equal(t, http.StatusCreated, status)

	// track1 has both tags; track2 only has "road".
	e.setTags(alice, track1["uuid"].(string), track1["name"].(string), []string{"alpine", "race"})
	e.setTags(alice, track2["uuid"].(string), track2["name"].(string), []string{"road"})

	// AND mode: must have both alpine AND race → only track1.
	var listResp map[string]any
	status, _ = e.do(alice, http.MethodGet, "/tracks?onlyMine=true&tag=alpine&tag=race&tagsAnd=true", nil, &listResp)
	assert.Equal(t, http.StatusOK, status)
	tracks, _ := listResp["tracks"].([]any)
	require.Len(t, tracks, 1)
	assert.Equal(t, track1["uuid"], tracks[0].(map[string]any)["uuid"])

	// AND mode: alpine AND road → no track has both → empty.
	status, _ = e.do(alice, http.MethodGet, "/tracks?onlyMine=true&tag=alpine&tag=road&tagsAnd=true", nil, &listResp)
	assert.Equal(t, http.StatusOK, status)
	tracks, _ = listResp["tracks"].([]any)
	assert.Empty(t, tracks)
}

func TestDeleteTrack_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)

	anon := e.newClient()
	status, _ = e.do(anon, http.MethodDelete, "/tracks/"+trackUUID, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestDeleteTrack_Forbidden(t *testing.T) {
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
	status, _ = e.do(bob, http.MethodDelete, "/tracks/"+trackUUID, nil, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestDeleteTrack_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)

	status, _ = e.do(alice, http.MethodDelete, "/tracks/"+trackUUID, nil, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// Track should no longer be accessible.
	status, _ = e.do(alice, http.MethodGet, "/tracks/"+trackUUID, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestDeleteTrack_NotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, _ := e.do(alice, http.MethodDelete, "/tracks/nonexistent-uuid", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackPoints_PublicTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)
	e.makeTrackPublic(alice, trackUUID, uploaded["name"].(string))

	anon := e.newClient()
	var resp map[string]any
	status, _ = e.do(anon, http.MethodGet, "/tracks/"+trackUUID+"/points", nil, &resp)
	assert.Equal(t, http.StatusOK, status)

	points, ok := resp["points"].([]any)
	require.True(t, ok, "response must contain points array")
	assert.Greater(t, len(points), 2, "subsampled track should have at least 3 points")

	// Each point must be an object with lat, lon, ele, d.
	first := points[0].(map[string]any)
	assert.IsType(t, float64(0), first["lat"])
	assert.IsType(t, float64(0), first["lon"])
	assert.IsType(t, float64(0), first["ele"])
	assert.IsType(t, float64(0), first["d"])
	assert.Equal(t, float64(0), first["d"], "first point should have zero cumulative distance")
}

func TestGetTrackPoints_PrivateTrack_Forbidden(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)

	anon := e.newClient()
	status, _ = e.do(anon, http.MethodGet, "/tracks/"+trackUUID+"/points", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackPoints_ETag(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)

	// First request: get the ETag.
	req, err := http.NewRequest(http.MethodGet, e.ts.URL+"/tracks/"+trackUUID+"/points", nil)
	require.NoError(t, err)
	resp, err := alice.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	eTag := resp.Header.Get("ETag")
	require.NotEmpty(t, eTag)

	// Second request with If-None-Match should return 304.
	req, err = http.NewRequest(http.MethodGet, e.ts.URL+"/tracks/"+trackUUID+"/points", nil)
	require.NoError(t, err)
	req.Header.Set("If-None-Match", eTag)
	resp, err = alice.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotModified, resp.StatusCode)
}

func TestGetTrack_IsOwner(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)
	e.makeTrackPublic(alice, trackUUID, uploaded["name"].(string))

	// Alice is the owner.
	var track map[string]any
	status, _ = e.do(alice, http.MethodGet, "/tracks/"+trackUUID, nil, &track)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, true, track["isOwner"])

	// Bob is not the owner.
	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")
	status, _ = e.do(bob, http.MethodGet, "/tracks/"+trackUUID, nil, &track)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, false, track["isOwner"])
}

// makeGPXWithName builds a minimal valid GPX with the given track name.
func makeGPXWithName(name string) []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" xmlns="http://www.topografix.com/GPX/1/1">
  <trk><name>` + name + `</name><trkseg>
    <trkpt lat="47.3769" lon="8.5417"><ele>408</ele></trkpt>
    <trkpt lat="47.3779" lon="8.5427"><ele>410</ele></trkpt>
    <trkpt lat="47.3789" lon="8.5437"><ele>412</ele></trkpt>
    <trkpt lat="47.3799" lon="8.5447"><ele>414</ele></trkpt>
    <trkpt lat="47.3809" lon="8.5457"><ele>416</ele></trkpt>
  </trkseg></trk>
</gpx>`)
}

func TestUploadTrack_NameFromMetadata(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, resp := e.doUploadRaw(client, makeGPXWithName("  My Track  "), "whatever.gpx")
	require.Equal(t, http.StatusCreated, status)
	assert.Equal(t, "My Track", resp["name"])
}

func TestUploadTrack_NameFallsBackToFilename(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, resp := e.doUploadRaw(client, makeGPXWithName("   "), "my-ride.gpx")
	require.Equal(t, http.StatusCreated, status)
	assert.Equal(t, "my-ride", resp["name"])
}

func TestUploadTrack_NameFallsBackToTimestamp(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, resp := e.doUploadRaw(client, makeGPXWithName(""), "  .gpx")
	require.Equal(t, http.StatusCreated, status)
	name, ok := resp["name"].(string)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(name, "Track uploaded "), "expected timestamp fallback, got: %s", name)
}
