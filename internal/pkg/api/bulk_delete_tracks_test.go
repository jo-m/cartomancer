package api_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkDeleteTracks_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPost, "/tracks/bulk-delete", map[string]any{
		"uuids": []string{"fake-uuid"},
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestBulkDeleteTracks_EmptyUUIDs(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodPost, "/tracks/bulk-delete", map[string]any{
		"uuids": []string{},
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestBulkDeleteTracks_NonexistentUUID(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodPost, "/tracks/bulk-delete", map[string]any{
		"uuids": []string{"nonexistent-uuid"},
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestBulkDeleteTracks_OtherUserTrack(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	e.createUser("bob@example.com", "Bob", "secret22", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret11")

	status, uploaded := e.doUpload(alice, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	trackUUID := uploaded["uuid"].(string)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret22")

	status, _ = e.do(bob, http.MethodPost, "/tracks/bulk-delete", map[string]any{
		"uuids": []string{trackUUID},
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestBulkDeleteTracks_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	// Upload two distinct tracks.
	status, u1 := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	uuid1 := u1["uuid"].(string)

	status, u2 := e.doUpload(client, testGPXFile2)
	require.Equal(t, http.StatusCreated, status)
	uuid2 := u2["uuid"].(string)

	// Bulk delete both tracks.
	status, _ = e.do(client, http.MethodPost, "/tracks/bulk-delete", map[string]any{
		"uuids": []string{uuid1, uuid2},
	}, nil)
	assert.Equal(t, http.StatusNoContent, status)

	// Verify tracks are gone.
	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid1, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status, _ = e.do(client, http.MethodGet, "/tracks/"+uuid2, nil, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestBulkDeleteTracks_TooManyUUIDs(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	uuids := make([]string, 501)
	for i := range uuids {
		uuids[i] = fmt.Sprintf("fake-uuid-%d", i)
	}

	status, _ := e.do(client, http.MethodPost, "/tracks/bulk-delete", map[string]any{
		"uuids": uuids,
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}
