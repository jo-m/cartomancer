package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"jo-m.ch/go/detour/internal/pkg/db"
)

func TestSearchGeonames(t *testing.T) {
	env := newTestEnv(t)
	client := env.newClient()

	// Seed test data.
	ctx := t.Context()
	q := env.d.QueryRW()

	require.NoError(t, q.InsertGeonameAdmin1(ctx, db.InsertGeonameAdmin1Params{
		Code: "CH.BE", Name: "Bern", Geonameid: 1,
	}))
	require.NoError(t, q.InsertGeonameAdmin2(ctx, db.InsertGeonameAdmin2Params{
		Code: "CH.BE.0246", Name: "Interlaken-Oberhasli", Geonameid: 2,
	}))
	require.NoError(t, q.InsertGeoname(ctx, db.InsertGeonameParams{
		Geonameid: 100, Name: "Bern", Asciiname: "Bern",
		Latitude: 46.94, Longitude: 7.45,
		FeatureClass: "P", FeatureCode: "PPLC",
		CountryCode: "CH", Admin1Code: "BE",
		Population: 130000,
	}))
	require.NoError(t, q.InsertGeoname(ctx, db.InsertGeonameParams{
		Geonameid: 101, Name: "Berne", Asciiname: "Berne",
		Latitude: 46.94, Longitude: 7.45,
		FeatureClass: "P", FeatureCode: "PPL",
		CountryCode: "CH", Admin1Code: "BE",
		Population: 1000,
	}))
	require.NoError(t, q.InsertGeoname(ctx, db.InsertGeonameParams{
		Geonameid: 102, Name: "Brig", Asciiname: "Brig",
		Latitude: 46.31, Longitude: 7.98,
		FeatureClass: "P", FeatureCode: "PPL",
		CountryCode: "CH", Admin1Code: "VS",
		Population: 13000,
	}))
	require.NoError(t, q.InsertGeoname(ctx, db.InsertGeonameParams{
		Geonameid: 103, Name: "Grindelwald", Asciiname: "Grindelwald",
		Latitude: 46.62, Longitude: 8.04,
		FeatureClass: "P", FeatureCode: "PPL",
		CountryCode: "CH", Admin1Code: "BE", Admin2Code: "0246",
		Population: 4000,
	}))
	require.NoError(t, q.InsertGeoname(ctx, db.InsertGeonameParams{
		Geonameid: 104, Name: "Zürich", Asciiname: "Zurich",
		Latitude: 47.37, Longitude: 8.55,
		FeatureClass: "P", FeatureCode: "PPLA",
		CountryCode: "CH", Admin1Code: "ZH",
		Population: 400000,
	}))
	// Non-place feature class (should not appear in results).
	require.NoError(t, q.InsertGeoname(ctx, db.InsertGeonameParams{
		Geonameid: 200, Name: "Bernina Pass", Asciiname: "Bernina Pass",
		Latitude: 46.41, Longitude: 10.03,
		FeatureClass: "T", FeatureCode: "PASS",
		CountryCode: "CH", Admin1Code: "GR",
	}))

	// Rebuild FTS index after seeding data.
	_, err := env.d.RW().ExecContext(ctx,
		`INSERT INTO geonames_fts(geonames_fts) VALUES('rebuild')`)
	require.NoError(t, err)

	_, err = env.d.QueryRW().CreateGeonameImport(ctx, db.CreateGeonameImportParams{
		CreatedAt: time.Now().UTC(),
		RowCount:  6,
	})
	require.NoError(t, err)

	t.Run("query too short", func(t *testing.T) {
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name?q=B", nil, nil)
		require.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("empty query", func(t *testing.T) {
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name", nil, nil)
		require.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("search prefix Ber", func(t *testing.T) {
		var resp geonameSearchResponseTest
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name?q=Ber", nil, &resp)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, resp.Results, 2)
		// Sorted by population descending: Bern (130000) before Berne (1000).
		require.Equal(t, "Bern", resp.Results[0].Name)
		require.Equal(t, "Bern", resp.Results[0].Admin1Name)
		require.Equal(t, "Berne", resp.Results[1].Name)
	})

	t.Run("search excludes non-places", func(t *testing.T) {
		var resp geonameSearchResponseTest
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name?q=Bernina", nil, &resp)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, resp.Results)
	})

	t.Run("admin2 joined", func(t *testing.T) {
		var resp geonameSearchResponseTest
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name?q=Grindelwald", nil, &resp)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, resp.Results, 1)
		require.Equal(t, "Grindelwald", resp.Results[0].Name)
		require.Equal(t, "Bern", resp.Results[0].Admin1Name)
		require.Equal(t, "Interlaken-Oberhasli", resp.Results[0].Admin2Name)
	})

	t.Run("accent insensitive", func(t *testing.T) {
		var resp geonameSearchResponseTest
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name?q=Z%C3%BCrich", nil, &resp)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, resp.Results, 1)
		require.Equal(t, "Zürich", resp.Results[0].Name)
	})

	t.Run("ascii matches accented name", func(t *testing.T) {
		var resp geonameSearchResponseTest
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name?q=Zurich", nil, &resp)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, resp.Results, 1)
		require.Equal(t, "Zürich", resp.Results[0].Name)
	})

	t.Run("case insensitive", func(t *testing.T) {
		var resp geonameSearchResponseTest
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name?q=bern", nil, &resp)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, resp.Results, 2)
	})

	t.Run("no results", func(t *testing.T) {
		var resp geonameSearchResponseTest
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name?q=Zzzzz", nil, &resp)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, resp.Results)
	})

	t.Run("FTS syntax injection safe", func(t *testing.T) {
		var resp geonameSearchResponseTest
		status, _ := env.do(client, http.MethodGet, "/geocode/search/name?q=OR+NOT", nil, &resp)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, resp.Results)
	})
}

// geonameSearchResponseTest mirrors the API response for test decoding.
type geonameSearchResponseTest struct {
	Results []geonameSearchResultTest `json:"results"`
}

type geonameSearchResultTest struct {
	GeonameID   int64   `json:"geonameId"`
	Name        string  `json:"name"`
	ASCIIName   string  `json:"asciiName"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	CountryCode string  `json:"countryCode"`
	FeatureCode string  `json:"featureCode"`
	Population  int64   `json:"population"`
	Admin1Name  string  `json:"admin1Name"`
	Admin2Name  string  `json:"admin2Name"`
}
