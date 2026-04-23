package api_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadAndGetUUID is a helper that uploads a track and returns its UUID.
func (e *testEnv) uploadAndGetUUID(client *http.Client) string {
	e.t.Helper()
	status, resp := e.doUpload(client, testGPXFile)
	require.Equal(e.t, http.StatusCreated, status)
	uuid, ok := resp["uuid"].(string)
	require.True(e.t, ok)
	return uuid
}

func TestSetTrackTags_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _ := e.do(client, http.MethodPut, "/tracks/fake-uuid/tags", []string{"tag1"}, nil)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestSetTrackTags_TrackNotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, _ := e.do(client, http.MethodPut, "/tracks/nonexistent/tags", []string{"tag1"}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestSetTrackTags_Forbidden(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	e.createUser("bob@example.com", "Bob", "secret22", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret11")
	trackUUID := e.uploadAndGetUUID(alice)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret22")

	status, _ := e.do(bob, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"tag1"}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestSetTrackTags_InvalidTag(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")
	trackUUID := e.uploadAndGetUUID(client)

	// Too short (1 char)
	status, _ := e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"x"}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	// Too long (33 chars)
	status, _ = e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"abcdefghijklmnopqrstuvwxyz1234567"}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	// Non-alphanumeric characters
	status, _ = e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"with-hyphen"}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status, _ = e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"with space"}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status, _ = e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"with_underscore"}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	// Unicode letters and digits are allowed
	status, _ = e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"velowege"}, nil)
	assert.Equal(t, http.StatusOK, status)
}

func TestSetTrackTags_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")
	trackUUID := e.uploadAndGetUUID(client)

	var resp map[string]any
	status, _ := e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"cycling", "weekend"}, &resp)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, trackUUID, resp["uuid"])

	tags, ok := resp["tags"].([]any)
	require.True(t, ok)
	assert.Len(t, tags, 2)
	assert.Contains(t, tags, "cycling")
	assert.Contains(t, tags, "weekend")
}

func TestSetTrackTags_ReplaceTags(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")
	trackUUID := e.uploadAndGetUUID(client)

	// Set initial tags
	status, _ := e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"oldtag"}, nil)
	require.Equal(t, http.StatusOK, status)

	// Replace with new tags
	var resp map[string]any
	status, _ = e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"newtag"}, &resp)
	assert.Equal(t, http.StatusOK, status)

	tags, ok := resp["tags"].([]any)
	require.True(t, ok)
	assert.Len(t, tags, 1)
	assert.Contains(t, tags, "newtag")
}

func TestSetTrackTags_EmptyList(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")
	trackUUID := e.uploadAndGetUUID(client)

	// Set some tags first
	status, _ := e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"cycling"}, nil)
	require.Equal(t, http.StatusOK, status)

	// Clear all tags
	var resp map[string]any
	status, _ = e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{}, &resp)
	assert.Equal(t, http.StatusOK, status)

	tags, ok := resp["tags"].([]any)
	require.True(t, ok)
	assert.Empty(t, tags)
}

func TestSuggestTags_Unauthenticated(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	// Unauthenticated requests receive an empty suggestion list rather than 401.
	var resp map[string]any
	status, _ := e.do(client, http.MethodGet, "/tags?prefix=cy", nil, &resp)
	assert.Equal(t, http.StatusOK, status)
	tags, _ := resp["tags"].([]any)
	assert.Empty(t, tags)
}

// extractTagNames pulls the tag names out of a suggestTags response body.
func extractTagNames(t *testing.T, resp map[string]any) []string {
	t.Helper()
	raw, ok := resp["tags"].([]any)
	require.True(t, ok)
	out := make([]string, len(raw))
	for i, item := range raw {
		m, ok := item.(map[string]any)
		require.True(t, ok)
		tag, ok := m["tag"].(string)
		require.True(t, ok)
		out[i] = tag
	}
	return out
}

func TestSuggestTags_NoPrefixReturnsAll(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")
	trackUUID := e.uploadAndGetUUID(client)

	status, _ := e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"cycling", "cityride", "commute"}, nil)
	require.Equal(t, http.StatusOK, status)

	var resp map[string]any
	status, _ = e.do(client, http.MethodGet, "/tags", nil, &resp)
	assert.Equal(t, http.StatusOK, status)

	names := extractTagNames(t, resp)
	assert.ElementsMatch(t, []string{"cycling", "cityride", "commute"}, names)
}

func TestSuggestTags_OrderedByTrackCount(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")

	status, r1 := e.doUpload(client, testGPXFile)
	require.Equal(t, http.StatusCreated, status)
	track1 := r1["uuid"].(string)
	status, r2 := e.doUpload(client, testGPXFile2)
	require.Equal(t, http.StatusCreated, status)
	track2 := r2["uuid"].(string)

	// "popular" on both tracks (count 2), "rare" on one (count 1).
	status, _ = e.do(client, http.MethodPut, "/tracks/"+track1+"/tags", []string{"popular", "rare"}, nil)
	require.Equal(t, http.StatusOK, status)
	status, _ = e.do(client, http.MethodPut, "/tracks/"+track2+"/tags", []string{"popular"}, nil)
	require.Equal(t, http.StatusOK, status)

	var resp map[string]any
	status, _ = e.do(client, http.MethodGet, "/tags", nil, &resp)
	assert.Equal(t, http.StatusOK, status)

	raw, ok := resp["tags"].([]any)
	require.True(t, ok)
	require.Len(t, raw, 2)

	first := raw[0].(map[string]any)
	second := raw[1].(map[string]any)

	assert.Equal(t, "popular", first["tag"])
	assert.EqualValues(t, 2, first["nTracks"])
	assert.Equal(t, "rare", second["tag"])
	assert.EqualValues(t, 1, second["nTracks"])
}

func TestSuggestTags_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")
	trackUUID := e.uploadAndGetUUID(client)

	status, _ := e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"cycling", "cityride", "commute"}, nil)
	require.Equal(t, http.StatusOK, status)

	var resp map[string]any
	status, _ = e.do(client, http.MethodGet, "/tags?prefix=cy", nil, &resp)
	assert.Equal(t, http.StatusOK, status)

	names := extractTagNames(t, resp)
	assert.Equal(t, []string{"cycling"}, names)
}

func TestSuggestTags_NoMatch(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret11")
	trackUUID := e.uploadAndGetUUID(client)

	status, _ := e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"cycling"}, nil)
	require.Equal(t, http.StatusOK, status)

	var resp map[string]any
	status, _ = e.do(client, http.MethodGet, "/tags?prefix=zz", nil, &resp)
	assert.Equal(t, http.StatusOK, status)

	tags, ok := resp["tags"].([]any)
	require.True(t, ok)
	assert.Empty(t, tags)
}

func TestSuggestTags_OtherUserTagsNotVisible(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)
	e.createUser("bob@example.com", "Bob", "secret22", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret11")
	trackUUID := e.uploadAndGetUUID(alice)

	status, _ := e.do(alice, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"cycling"}, nil)
	require.Equal(t, http.StatusOK, status)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret22")

	var resp map[string]any
	status, _ = e.do(bob, http.MethodGet, "/tags?prefix=cy", nil, &resp)
	assert.Equal(t, http.StatusOK, status)

	tags, ok := resp["tags"].([]any)
	require.True(t, ok)
	assert.Empty(t, tags)
}
