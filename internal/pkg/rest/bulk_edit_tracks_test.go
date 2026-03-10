package rest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkEditTracks_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPatch, "/tracks", map[string]any{
		"uuids": []string{"fake-uuid"},
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestBulkEditTracks_EmptyUUIDs(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodPatch, "/tracks", map[string]any{
		"uuids": []string{},
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestBulkEditTracks_NotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodPatch, "/tracks", map[string]any{
		"uuids":  []string{"nonexistent-uuid"},
		"public": true,
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestBulkEditTracks_OtherUserTrack(t *testing.T) {
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

	status, _ = e.do(bob, http.MethodPatch, "/tracks", map[string]any{
		"uuids":  []string{trackUUID},
		"public": true,
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestBulkEditTracks_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	// Upload two tracks.
	status, u1 := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid1 := u1["uuid"].(string)

	status, u2 := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid2 := u2["uuid"].(string)

	// Bulk edit both tracks.
	status, _ = e.do(client, http.MethodPatch, "/tracks", map[string]any{
		"uuids":  []string{uuid1, uuid2},
		"public": true,
		"source": "strava",
		"sport":  2,
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// Verify changes on track 1.
	var track1 map[string]any
	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid1, nil, &track1)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, true, track1["public"])
	assert.Equal(t, "strava", track1["source"])
	assert.Equal(t, float64(2), track1["sport"])

	// Verify changes on track 2.
	var track2 map[string]any
	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid2, nil, &track2)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, true, track2["public"])
	assert.Equal(t, "strava", track2["source"])
	assert.Equal(t, float64(2), track2["sport"])
}

func TestBulkEditTracks_PartialFields(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, uploaded := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)

	originalName := uploaded["name"].(string)

	// Only update author — other fields should remain unchanged.
	status, _ = e.do(client, http.MethodPatch, "/tracks", map[string]any{
		"uuids":  []string{trackUUID},
		"author": "Jane",
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	var track map[string]any
	status, _ = e.do(client, http.MethodGet, "/tracks/"+trackUUID, nil, &track)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Jane", track["author"])
	assert.Equal(t, originalName, track["name"])
}

func TestBulkEditTracks_Tags(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	// Upload two tracks.
	status, u1 := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid1 := u1["uuid"].(string)

	status, u2 := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid2 := u2["uuid"].(string)

	// Bulk set tags on both tracks.
	status, _ = e.do(client, http.MethodPatch, "/tracks", map[string]any{
		"uuids": []string{uuid1, uuid2},
		"tags":  []string{"cycling", "road"},
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// Verify tags on track 1.
	var track1 map[string]any
	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid1, nil, &track1)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, []any{"cycling", "road"}, track1["tags"])

	// Verify tags on track 2.
	var track2 map[string]any
	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid2, nil, &track2)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, []any{"cycling", "road"}, track2["tags"])

	// Replace tags with a different set (set, not add).
	status, _ = e.do(client, http.MethodPatch, "/tracks", map[string]any{
		"uuids": []string{uuid1, uuid2},
		"tags":  []string{"hiking"},
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid1, nil, &track1)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, []any{"hiking"}, track1["tags"])

	// Clear tags by setting an empty list.
	status, _ = e.do(client, http.MethodPatch, "/tracks", map[string]any{
		"uuids": []string{uuid1},
		"tags":  []string{},
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid1, nil, &track1)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, []any{}, track1["tags"])
}

func TestBulkEditTracks_NoFieldsProvided(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, uploaded := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)

	// No fields besides uuids — should be a no-op success.
	status, _ = e.do(client, http.MethodPatch, "/tracks", map[string]any{
		"uuids": []string{trackUUID},
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)
}
