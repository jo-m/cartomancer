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
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodPut, "/tracks/nonexistent/tags", []string{"tag1"}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestSetTrackTags_Forbidden(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadAndGetUUID(alice)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")

	status, _ := e.do(bob, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"tag1"}, nil)
	assert.Equal(t, http.StatusForbidden, status)
}

func TestSetTrackTags_InvalidTag(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")
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
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")
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
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")
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
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")
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

func TestSuggestTags_PrefixTooShort(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")

	status, _ := e.do(client, http.MethodGet, "/tags?prefix=c", nil, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestSuggestTags_Success(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")
	trackUUID := e.uploadAndGetUUID(client)

	// Create some tags
	status, _ := e.do(client, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"cycling", "cityride", "commute"}, nil)
	require.Equal(t, http.StatusOK, status)

	// Suggest with matching prefix
	var resp map[string]any
	status, _ = e.do(client, http.MethodGet, "/tags?prefix=cy", nil, &resp)
	assert.Equal(t, http.StatusOK, status)

	tags, ok := resp["tags"].([]any)
	require.True(t, ok)
	assert.Len(t, tags, 1)
	assert.Equal(t, "cycling", tags[0])
}

func TestSuggestTags_NoMatch(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret", false)
	client := e.newClient()
	e.login(client, "alice@example.com", "secret")
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
	e.createUser("alice@example.com", "Alice", "secret", false)
	e.createUser("bob@example.com", "Bob", "secret2", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret")
	trackUUID := e.uploadAndGetUUID(alice)

	status, _ := e.do(alice, http.MethodPut, "/tracks/"+trackUUID+"/tags", []string{"cycling"}, nil)
	require.Equal(t, http.StatusOK, status)

	bob := e.newClient()
	e.login(bob, "bob@example.com", "secret2")

	var resp map[string]any
	status, _ = e.do(bob, http.MethodGet, "/tags?prefix=cy", nil, &resp)
	assert.Equal(t, http.StatusOK, status)

	tags, ok := resp["tags"].([]any)
	require.True(t, ok)
	assert.Empty(t, tags)
}
