package rest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadPublicTrack uploads testGPXFile and makes it public.
// Returns the track UUID.
func (e *testEnv) uploadPublicTrack(client *http.Client) string {
	e.t.Helper()
	status, resp := e.doUpload(client, testGPXFile)
	require.Equal(e.t, http.StatusCreated, status)
	trackUUID, ok := resp["uuid"].(string)
	require.True(e.t, ok)
	e.makeTrackPublic(client, trackUUID, resp["name"].(string))
	return trackUUID
}

// uploadPrivateTrack uploads testGPXFile2 and leaves it private.
// Returns the track UUID.
func (e *testEnv) uploadPrivateTrack(client *http.Client) string {
	e.t.Helper()
	status, resp := e.doUpload(client, testGPXFile2)
	require.Equal(e.t, http.StatusCreated, status)
	trackUUID, ok := resp["uuid"].(string)
	require.True(e.t, ok)
	return trackUUID
}

func TestStarTrack_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPublicTrack(alice)

	anon := e.newClient()
	status, _ := e.do(anon, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestStarTrack_PublicTrack(t *testing.T) {
	e := newTestEnv(t)
	aliceUUID := e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPublicTrack(alice)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")

	// Bob stars alice's public track.
	status, _ := e.do(bob, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// Alice's star list as seen by alice contains the track.
	var stars []any
	status, _ = e.do(alice, http.MethodGet, "/users/"+aliceUUID+"/stars", nil, nil)
	// Alice has not starred anything.
	assert.Equal(t, http.StatusOK, status)

	// Bob's star list contains alice's track.
	bobUUID, err := e.d.QueryRO().GetUserByEmail(t.Context(), "bob@example.com")
	require.NoError(t, err)
	status, _ = e.do(alice, http.MethodGet, "/users/"+bobUUID.Uuid+"/stars", nil, &stars)
	assert.Equal(t, http.StatusOK, status)
	assert.Len(t, stars, 1)
}

func TestStarTrack_OwnPrivateTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPrivateTrack(alice)

	status, _ := e.do(alice, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestStarTrack_OtherUserPrivateTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPrivateTrack(alice)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")

	// Bob cannot star alice's private track.
	status, _ := e.do(bob, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestStarTrack_Idempotent(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPublicTrack(alice)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")

	// Starring twice should both succeed.
	status, _ := e.do(bob, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	assert.Equal(t, http.StatusNoContent, status)
	status, _ = e.do(bob, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestUnstarTrack_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPublicTrack(alice)

	status, _ := e.do(alice, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)

	status, _ = e.do(alice, http.MethodDelete, "/tracks/"+trackUUID+"/star", nil, nil)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestUnstarTrack_NotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPublicTrack(alice)

	// Alice never starred this track, so unstarring returns 404.
	status, _ := e.do(alice, http.MethodDelete, "/tracks/"+trackUUID+"/star", nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetUserStars_OwnerSeesPrivateStars(t *testing.T) {
	e := newTestEnv(t)
	aliceUUID := e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	publicTrack := e.uploadPublicTrack(alice)
	privateTrack := e.uploadPrivateTrack(alice)

	// Alice stars both her own tracks.
	status, _ := e.do(alice, http.MethodPost, "/tracks/"+publicTrack+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)
	status, _ = e.do(alice, http.MethodPost, "/tracks/"+privateTrack+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)

	var stars []any
	status, _ = e.do(alice, http.MethodGet, "/users/"+aliceUUID+"/stars", nil, &stars)
	assert.Equal(t, http.StatusOK, status)
	assert.Len(t, stars, 2)
}

func TestGetUserStars_OtherUserSeesOnlyPublicStars(t *testing.T) {
	e := newTestEnv(t)
	aliceUUID := e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	publicTrack := e.uploadPublicTrack(alice)
	privateTrack := e.uploadPrivateTrack(alice)

	status, _ := e.do(alice, http.MethodPost, "/tracks/"+publicTrack+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)
	status, _ = e.do(alice, http.MethodPost, "/tracks/"+privateTrack+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")

	var stars []any
	status, _ = e.do(bob, http.MethodGet, "/users/"+aliceUUID+"/stars", nil, &stars)
	assert.Equal(t, http.StatusOK, status)
	assert.Len(t, stars, 1)
}

func TestGetUserStars_AnonymousSeesOnlyPublicStars(t *testing.T) {
	e := newTestEnv(t)
	aliceUUID := e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")

	publicTrack := e.uploadPublicTrack(alice)
	privateTrack := e.uploadPrivateTrack(alice)

	status, _ := e.do(alice, http.MethodPost, "/tracks/"+publicTrack+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)
	status, _ = e.do(alice, http.MethodPost, "/tracks/"+privateTrack+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)

	anon := e.newClient()

	var stars []any
	status, _ = e.do(anon, http.MethodGet, "/users/"+aliceUUID+"/stars", nil, &stars)
	assert.Equal(t, http.StatusOK, status)
	assert.Len(t, stars, 1)
}

func TestGetUserStars_Empty(t *testing.T) {
	e := newTestEnv(t)
	aliceUUID := e.createUser("alice@example.com", "Alice", "secret", false)

	anon := e.newClient()

	var stars []any
	status, _ := e.do(anon, http.MethodGet, "/users/"+aliceUUID+"/stars", nil, &stars)
	assert.Equal(t, http.StatusOK, status)
	assert.Empty(t, stars)
}

func TestIsStarred_ReturnedOnGetTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPublicTrack(alice)

	// Before starring: isStarred should be false.
	var resp map[string]any
	status, _ := e.do(alice, http.MethodGet, "/tracks/"+trackUUID, nil, &resp)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, false, resp["isStarred"])

	// After starring: isStarred should be true.
	status, _ = e.do(alice, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)

	status, _ = e.do(alice, http.MethodGet, "/tracks/"+trackUUID, nil, &resp)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, true, resp["isStarred"])
}

func TestIsStarred_ReturnedOnListTracks(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPublicTrack(alice)

	status, _ := e.do(alice, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)

	var listResp map[string]any
	status, _ = e.do(alice, http.MethodGet, "/tracks", nil, &listResp)
	require.Equal(t, http.StatusOK, status)

	tracks := listResp["tracks"].([]any)
	require.Len(t, tracks, 1)
	tr := tracks[0].(map[string]any)
	assert.Equal(t, true, tr["isStarred"])
}

func TestIsStarred_FalseForAnonymousOnGetTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPublicTrack(alice)

	// Alice stars the track.
	status, _ := e.do(alice, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)

	// Anonymous viewer should see isStarred = false.
	anon := e.newClient()
	var resp map[string]any
	status, _ = e.do(anon, http.MethodGet, "/tracks/"+trackUUID, nil, &resp)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, false, resp["isStarred"])
}

func TestIsStarred_TrueOnUserStarsList(t *testing.T) {
	e := newTestEnv(t)
	aliceUUID := e.createUser("alice@example.com", "Alice", "secret", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadPublicTrack(alice)

	status, _ := e.do(alice, http.MethodPost, "/tracks/"+trackUUID+"/star", nil, nil)
	require.Equal(t, http.StatusNoContent, status)

	var stars []map[string]any
	status, _ = e.do(alice, http.MethodGet, "/users/"+aliceUUID+"/stars", nil, &stars)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, stars, 1)
	assert.Equal(t, true, stars[0]["isStarred"])
}
