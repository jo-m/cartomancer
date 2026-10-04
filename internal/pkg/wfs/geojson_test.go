package wfs

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"
)

// featureJSON renders one GeoJSON Point feature with a properties object.
func featureJSON(id, nummer string) string {
	return fmt.Sprintf(
		`{"type":"Feature","id":%q,"properties":{"nummer":%q},"geometry":{"type":"Point","coordinates":[8.44,47.34]}}`,
		id, nummer,
	)
}

// newPageServer serves the given pages keyed by the startIndex query
// parameter and records every request's query parameters. Pages not present
// in the map are served as empty FeatureCollections.
func newPageServer(t *testing.T, pages map[string]string) (*httptest.Server, *[]url.Values) {
	t.Helper()
	requests := &[]url.Values{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		*requests = append(*requests, q)
		w.Header().Set("Content-Type", "application/json")
		body, ok := pages[q.Get("startIndex")]
		if !ok {
			body = `{"type":"FeatureCollection","features":[]}`
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, requests
}

func TestGetFeatureGeoJSONPaginates(t *testing.T) {
	srv, requests := newPageServer(t, map[string]string{
		"0": `{"type":"FeatureCollection","features":[` + featureJSON("f1", "1") + `,` + featureJSON("f2", "2") + `]}`,
		"2": `{"type":"FeatureCollection","features":[` + featureJSON("f3", "3") + `]}`,
	})

	c := NewClient(srv.URL)
	features, err := c.GetFeatureGeoJSON(t.Context(), GetFeatureParams{
		TypeNames: "ms:test",
		Count:     2,
		SRSName:   "urn:ogc:def:crs:EPSG::4326",
	})
	require.NoError(t, err)
	require.Len(t, features, 3)
	require.Equal(t, "f1", features[0].ID)
	require.Equal(t, "f3", features[2].ID)
	require.JSONEq(t, `{"nummer":"1"}`, string(features[0].Properties))
	require.NotNil(t, features[0].Geometry)
	pt, ok := features[0].Geometry.Geometry().(orb.Point)
	require.True(t, ok, "expected a point geometry, got %T", features[0].Geometry.Geometry())
	require.InDelta(t, 8.44, pt.Lon(), 1e-9)
	require.InDelta(t, 47.34, pt.Lat(), 1e-9)

	// Two pages of data plus one empty page that terminates the loop.
	// startIndex advances by the number of features actually returned.
	require.Len(t, *requests, 3)
	for i, q := range *requests {
		require.Equal(t, "GetFeature", q.Get("request"))
		require.Equal(t, "WFS", q.Get("service"))
		require.Equal(t, "2.0.0", q.Get("version"))
		require.Equal(t, "ms:test", q.Get("typeNames"))
		require.Empty(t, q.Get("typeName"))
		require.Equal(t, "2", q.Get("count"))
		require.Empty(t, q.Get("maxFeatures"))
		require.Equal(t, []string{"0", "2", "3"}[i], q.Get("startIndex"))
		require.Equal(t, "application/json", q.Get("outputFormat"))
		require.Equal(t, "urn:ogc:def:crs:EPSG::4326", q.Get("srsName"))
	}
}

func TestGetFeatureGeoJSONV11Params(t *testing.T) {
	srv, requests := newPageServer(t, map[string]string{
		"0": `{"type":"FeatureCollection","features":[` + featureJSON("f1", "1") + `]}`,
	})

	c := NewClientV11(srv.URL)
	features, err := c.GetFeatureGeoJSON(t.Context(), GetFeatureParams{
		TypeNames: "ewz_brennstelle_p",
		Count:     1,
		SRSName:   "EPSG:4326",
	})
	require.NoError(t, err)
	require.Len(t, features, 1)

	require.Len(t, *requests, 2)
	for _, q := range *requests {
		// WFS 1.1 uses different parameter names and a different GeoJSON
		// MIME type than 2.0.
		require.Equal(t, "1.1.0", q.Get("version"))
		require.Equal(t, "ewz_brennstelle_p", q.Get("typeName"))
		require.Empty(t, q.Get("typeNames"))
		require.Equal(t, "1", q.Get("maxFeatures"))
		require.Empty(t, q.Get("count"))
		require.Equal(t, "application/vnd.geo+json", q.Get("outputFormat"))
		require.Equal(t, "EPSG:4326", q.Get("srsName"))
	}
	require.Equal(t, "0", (*requests)[0].Get("startIndex"))
	require.Equal(t, "1", (*requests)[1].Get("startIndex"))
}

func TestGetFeatureGeoJSONEmptyFirstPage(t *testing.T) {
	srv, requests := newPageServer(t, nil)

	c := NewClient(srv.URL)
	features, err := c.GetFeatureGeoJSON(t.Context(), GetFeatureParams{
		TypeNames: "ms:empty",
		Count:     10,
	})
	require.NoError(t, err)
	require.Empty(t, features)
	require.Len(t, *requests, 1)
}

func TestGetFeatureGeoJSONWithoutGeometry(t *testing.T) {
	srv, _ := newPageServer(t, map[string]string{
		"0": `{"type":"FeatureCollection","features":[{"type":"Feature","id":"f1","properties":{"a":1}}]}`,
	})

	c := NewClient(srv.URL)
	features, err := c.GetFeatureGeoJSON(t.Context(), GetFeatureParams{
		TypeNames: "ms:test",
		Count:     10,
	})
	require.NoError(t, err)
	require.Len(t, features, 1)
	require.Nil(t, features[0].Geometry)
	require.Equal(t, "f1", features[0].ID)
}

func TestGetFeatureGeoJSONArgumentValidation(t *testing.T) {
	c := NewClient("https://example.com/wfs")

	_, err := c.GetFeatureGeoJSON(t.Context(), GetFeatureParams{Count: 10})
	require.ErrorContains(t, err, "requires TypeNames")

	_, err = c.GetFeatureGeoJSON(t.Context(), GetFeatureParams{TypeNames: "x"})
	require.ErrorContains(t, err, "positive Count")
}

func TestGetFeatureGeoJSONExceptionReport(t *testing.T) {
	body := readFixture(t, "exception.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// MapServer returns exception reports with HTTP 200.
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL)
	_, err := c.GetFeatureGeoJSON(t.Context(), GetFeatureParams{
		TypeNames: "ms:does-not-exist",
		Count:     10,
	})
	var exc *ExceptionReport
	require.ErrorAs(t, err, &exc)
	require.NotEmpty(t, exc.Exceptions)
}

func TestGetFeatureGeoJSONNonJSONResponses(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		errContain string
	}{
		{
			name:       "error status keeps body",
			status:     http.StatusInternalServerError,
			body:       "<html>boom</html>",
			errContain: "unexpected status 500",
		},
		{
			name:       "ok status with html",
			status:     http.StatusOK,
			body:       "<html>boom not json</html>",
			errContain: "unexpected non-JSON response",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(srv.Close)

			c := NewClient(srv.URL)
			_, err := c.GetFeatureGeoJSON(t.Context(), GetFeatureParams{
				TypeNames: "ms:test",
				Count:     10,
			})
			require.ErrorContains(t, err, tc.errContain)
			// The response body must be part of the error, it is often the
			// only diagnostic for malformed requests.
			require.ErrorContains(t, err, "boom")
		})
	}
}

func TestUserAgent(t *testing.T) {
	got := make([]string, 0, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Header.Get("User-Agent"))
		if r.URL.Query().Get("outputFormat") != "" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"type":"FeatureCollection","features":[]}`))
			return
		}
		_, _ = w.Write(readFixture(t, "features_empty.xml"))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL)
	_, err := c.GetFeature(t.Context(), GetFeatureParams{TypeNames: "ms:test"})
	require.NoError(t, err)

	_, err = c.GetFeatureGeoJSON(t.Context(), GetFeatureParams{TypeNames: "ms:test", Count: 10})
	require.NoError(t, err)

	// Some servers reject Go's default user agent with a 403, so both
	// request paths must send the package's own user agent.
	require.Equal(t, []string{userAgent, userAgent}, got)
}

func TestNewClientV11(t *testing.T) {
	c := NewClientV11("https://example.com/wfs/")
	require.Equal(t, "https://example.com/wfs", c.baseURL)
	require.Equal(t, Version11, c.version)

	u := c.requestURL(url.Values{"request": {"GetFeature"}})
	// url.Values.Encode sorts keys alphabetically, so the order is stable.
	require.Equal(t,
		"https://example.com/wfs?request=GetFeature&service=WFS&version=1.1.0",
		u,
	)
}

func TestXMLPathsRejectV11(t *testing.T) {
	c := NewClientV11("https://example.com/wfs")

	_, err := c.GetFeature(t.Context(), GetFeatureParams{TypeNames: "x"})
	require.ErrorContains(t, err, "GetFeature requires WFS 2.0.0")

	_, err = c.GetCapabilities(t.Context())
	require.ErrorContains(t, err, "GetCapabilities requires WFS 2.0.0")
}
