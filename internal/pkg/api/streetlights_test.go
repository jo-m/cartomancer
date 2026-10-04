package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"jo-m.ch/go/cartomancer/internal/pkg/attribute"
	"jo-m.ch/go/cartomancer/internal/pkg/db"
	"jo-m.ch/go/cartomancer/internal/pkg/streetlights"
	"jo-m.ch/go/cartomancer/internal/pkg/track"
)

// apiTestAttribution is the source credit used by the streetlight endpoint
// tests.
var apiTestAttribution = attribute.Attribution{Author: "Test Office", Source: "https://example.com/lamps"}

// insertTrackLamp inserts a streetlight on the track's dp5m polyline, at the
// first vertex at or after the given fraction of the track length.
func insertTrackLamp(t *testing.T, d *db.DB, trackUUID string, fraction float64) orb.Point {
	t.Helper()
	ctx := t.Context()

	row, err := d.QueryRO().GetTrackByUUID(ctx, trackUUID)
	require.NoError(t, err)
	pts, err := track.DecodeVarint(row.PolylineDp5mVarint)
	require.NoError(t, err)
	require.NotEmpty(t, pts)

	targetM := pts[len(pts)-1].Distance * fraction
	ll := orb.Point{pts[len(pts)-1].Lon, pts[len(pts)-1].Lat}
	for _, p := range pts {
		if p.Distance >= targetM {
			ll = orb.Point{p.Lon, p.Lat}
			break
		}
	}

	err = d.WithTx(ctx, func(tx *db.Queries) error {
		return streetlights.Insert(ctx, tx, streetlights.StreetlightInsert{
			SourceID:    "api-test-lamp",
			InsertedBy:  "test",
			Geometry:    geojson.NewGeometry(ll),
			Attribution: apiTestAttribution,
		}, time.Now())
	})
	require.NoError(t, err)
	return ll
}

// getTrackStreetlights performs GET /tracks/{uuid}/streetlights, optionally
// with an If-None-Match header, and returns the status, the response ETag,
// and the decoded body for 200 responses.
func getTrackStreetlights(t *testing.T, e *testEnv, client *http.Client, trackUUID, ifNoneMatch string) (int, string, map[string]any) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, e.ts.URL+"/tracks/"+trackUUID+"/streetlights", nil)
	require.NoError(t, err)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var decoded map[string]any
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.Unmarshal(body, &decoded))
	}
	return resp.StatusCode, resp.Header.Get("ETag"), decoded
}

func TestGetTrackStreetlights_NotFound(t *testing.T) {
	e := newTestEnv(t)
	client := e.newClient()

	status, _, _ := getTrackStreetlights(t, e, client, "nonexistent", "")
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackStreetlights_PrivateTrackAnonymous(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret11")
	trackUUID := e.uploadPrivateTrack(alice)

	anon := e.newClient()
	status, _, _ := getTrackStreetlights(t, e, anon, trackUUID, "")
	assert.Equal(t, http.StatusNotFound, status)
}

func TestGetTrackStreetlights_NoLamps(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret11")
	trackUUID := e.uploadPublicTrack(alice)

	status, _, body := getTrackStreetlights(t, e, alice, trackUUID, "")
	require.Equal(t, http.StatusOK, status)
	assert.Greater(t, body["totalDistanceM"], float64(0))
	assert.Equal(t, float64(0), body["litDistanceM"])
	assert.Equal(t, float64(0), body["litFraction"])
	assert.Equal(t, float64(5), body["sampleStepM"])
	assert.Equal(t, float64(20), body["litRadiusM"])
	assert.Equal(t, []any{}, body["stretches"])
	assert.Equal(t, []any{}, body["attributions"])
}

func TestGetTrackStreetlights_LitAndAttribution(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret11")
	trackUUID := e.uploadPublicTrack(alice)

	insertTrackLamp(t, e.d, trackUUID, 0.5)

	status, _, body := getTrackStreetlights(t, e, alice, trackUUID, "")
	require.Equal(t, http.StatusOK, status)

	// A single lamp lights a short section of a long track.
	litDistanceM := body["litDistanceM"].(float64)
	assert.Positive(t, litDistanceM)
	assert.Less(t, litDistanceM, body["totalDistanceM"].(float64))
	assert.Greater(t, body["litFraction"].(float64), float64(0))
	assert.Less(t, body["litFraction"].(float64), float64(1))

	stretches := body["stretches"].([]any)
	require.NotEmpty(t, stretches)
	first := stretches[0].(map[string]any)
	assert.GreaterOrEqual(t, first["startDistanceM"].(float64), float64(0))
	assert.Less(t, first["startDistanceM"].(float64), first["endDistanceM"].(float64))

	assert.Equal(t, []any{
		map[string]any{"text": apiTestAttribution.Author, "href": apiTestAttribution.Source},
	}, body["attributions"])
}

func TestGetTrackStreetlights_ETag(t *testing.T) {
	e := newTestEnv(t)
	e.createUser("alice@example.com", "Alice", "secret11", false)

	alice := e.newClient()
	e.login(alice, "alice@example.com", "secret11")
	trackUUID := e.uploadPublicTrack(alice)

	status, eTag, _ := getTrackStreetlights(t, e, alice, trackUUID, "")
	require.Equal(t, http.StatusOK, status)
	require.NotEmpty(t, eTag)

	// The same track and lamp corpus must validate to 304.
	status, _, _ = getTrackStreetlights(t, e, alice, trackUUID, eTag)
	assert.Equal(t, http.StatusNotModified, status)

	// Inserting a lamp must change the validator.
	insertTrackLamp(t, e.d, trackUUID, 0.5)
	status, newETag, _ := getTrackStreetlights(t, e, alice, trackUUID, eTag)
	assert.Equal(t, http.StatusOK, status)
	assert.NotEqual(t, eTag, newETag)
}
