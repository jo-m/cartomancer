package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/db"
)

func createTestTrack(t *testing.T, d *db.DB, userID, name string) string {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()

	blob, err := d.QueryRW().CreateBlob(t.Context(), db.CreateBlobParams{
		Compression: 0,
		Content:     []byte("test"),
		HashType:    0,
		Hash:        []byte("test"),
	})
	require.NoError(t, err)

	_, err = d.QueryRW().CreateTrack(t.Context(), db.CreateTrackParams{
		Uuid:             id.String(),
		CreatedAt:        now,
		UpdatedAt:        now,
		UserID:           userID,
		BlobID:           blob.ID,
		FileFormat:       0,
		OriginalFilename: "test.gpx",
		Name:             name,
		TotalDistanceM:   10000,
		TotalAscentM:     500,
	})
	require.NoError(t, err)
	return id.String()
}

func createTestGroup(t *testing.T, d *db.DB, userID string, trackIDs []string) string {
	t.Helper()
	id, err := uuid.NewV7()
	require.NoError(t, err)
	now := time.Now().UTC()
	err = d.QueryRW().CreateTrackGroup(t.Context(), db.CreateTrackGroupParams{
		Uuid:      id.String(),
		CreatedAt: now,
		UserID:    userID,
	})
	require.NoError(t, err)
	for _, tid := range trackIDs {
		err = d.QueryRW().CreateTrackGroupMember(t.Context(), db.CreateTrackGroupMemberParams{
			GroupID: id.String(),
			TrackID: tid,
		})
		require.NoError(t, err)
	}
	return id.String()
}

func TestListTrackGroups_RequiresAuth(t *testing.T) {
	e := newTestEnv(t)
	c := e.newClient()
	status, _ := e.do(c, http.MethodGet, "/tracks/groups", nil, nil)
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestListTrackGroups_Empty(t *testing.T) {
	e := newTestEnv(t)
	userID := e.createUser("u@test.com", "User", "pass1234", false)
	c := e.newClient()
	e.login(c, "u@test.com", "pass1234")

	var resp struct {
		Groups []struct {
			UUID        string `json:"uuid"`
			MemberCount int    `json:"memberCount"`
			SampleName  string `json:"sampleName"`
		} `json:"groups"`
	}
	status, _ := e.do(c, http.MethodGet, "/tracks/groups", nil, &resp)
	require.Equal(t, http.StatusOK, status)
	require.Empty(t, resp.Groups)

	// Group with only 1 member should be filtered out.
	t1 := createTestTrack(t, e.d, userID, "Solo Track")
	createTestGroup(t, e.d, userID, []string{t1})

	status, _ = e.do(c, http.MethodGet, "/tracks/groups", nil, &resp)
	require.Equal(t, http.StatusOK, status)
	require.Empty(t, resp.Groups)
}

func TestListTrackGroups_ReturnsMultiMemberGroups(t *testing.T) {
	e := newTestEnv(t)
	userID := e.createUser("u@test.com", "User", "pass1234", false)
	c := e.newClient()
	e.login(c, "u@test.com", "pass1234")

	t1 := createTestTrack(t, e.d, userID, "Alpha Route")
	t2 := createTestTrack(t, e.d, userID, "Beta Route")
	groupID := createTestGroup(t, e.d, userID, []string{t1, t2})

	var resp struct {
		Groups []struct {
			UUID        string `json:"uuid"`
			MemberCount int    `json:"memberCount"`
			SampleName  string `json:"sampleName"`
		} `json:"groups"`
	}
	status, _ := e.do(c, http.MethodGet, "/tracks/groups", nil, &resp)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, resp.Groups, 1)
	require.Equal(t, groupID, resp.Groups[0].UUID)
	require.Equal(t, 2, resp.Groups[0].MemberCount)
	require.Equal(t, "Alpha Route", resp.Groups[0].SampleName)
}

func TestGetTrackGroup_RequiresAuth(t *testing.T) {
	e := newTestEnv(t)
	c := e.newClient()
	status, _ := e.do(c, http.MethodGet, "/tracks/groups/some-id", nil, nil)
	require.Equal(t, http.StatusUnauthorized, status)
}

func TestGetTrackGroup_NotFound(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("u@test.com", "User", "pass1234", false)
	c := e.newClient()
	e.login(c, "u@test.com", "pass1234")

	status, _ := e.do(c, http.MethodGet, "/tracks/groups/nonexistent", nil, nil)
	require.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackGroup_OtherUserForbidden(t *testing.T) {
	e := newTestEnv(t)
	ownerID := e.createUser("owner@test.com", "Owner", "pass1234", false)
	e.createUser("other@test.com", "Other", "pass1234", false)

	t1 := createTestTrack(t, e.d, ownerID, "Track A")
	t2 := createTestTrack(t, e.d, ownerID, "Track B")
	groupID := createTestGroup(t, e.d, ownerID, []string{t1, t2})

	c := e.newClient()
	e.login(c, "other@test.com", "pass1234")
	status, _ := e.do(c, http.MethodGet, "/tracks/groups/"+groupID, nil, nil)
	require.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackGroup_ReturnsMemberTracks(t *testing.T) {
	e := newTestEnv(t)
	userID := e.createUser("u@test.com", "User", "pass1234", false)
	c := e.newClient()
	e.login(c, "u@test.com", "pass1234")

	t1 := createTestTrack(t, e.d, userID, "Alpha Route")
	t2 := createTestTrack(t, e.d, userID, "Beta Route")
	groupID := createTestGroup(t, e.d, userID, []string{t1, t2})

	var resp struct {
		UUID   string `json:"uuid"`
		Tracks []struct {
			UUID string `json:"uuid"`
			Name string `json:"name"`
		} `json:"tracks"`
	}
	status, _ := e.do(c, http.MethodGet, "/tracks/groups/"+groupID, nil, &resp)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, groupID, resp.UUID)
	require.Len(t, resp.Tracks, 2)

	names := []string{resp.Tracks[0].Name, resp.Tracks[1].Name}
	require.Contains(t, names, "Alpha Route")
	require.Contains(t, names, "Beta Route")
}
