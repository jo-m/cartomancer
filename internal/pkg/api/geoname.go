package api

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"jo-m.ch/go/detour/internal/pkg/db"
	"jo-m.ch/go/detour/internal/pkg/logg"
)

const (
	geonameSearchMinQuery   = 2
	geonameSearchMaxResults = 20
)

type geonameSearchResult struct {
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

type geonameSearchResponse struct {
	Results []geonameSearchResult `json:"results"`
}

// handleSearchGeonames searches populated places by name prefix and returns
// results with resolved admin1/admin2 names.
func (sv *server) handleSearchGeonames(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(q) < geonameSearchMinQuery {
		writeError(w, http.StatusBadRequest, "query must be at least 2 characters")
		return
	}

	escaped := strings.NewReplacer("%", `\%`, "_", `\_`).Replace(q)
	rows, err := sv.d.QueryRO().SearchGeonames(ctx, db.SearchGeonamesParams{
		Query:      escaped + "%",
		MaxResults: geonameSearchMaxResults,
	})
	if err != nil {
		logg.Error(ctx, "failed to search geonames", "err", err)
		writeStatusError(w, http.StatusInternalServerError)
		return
	}

	results := make([]geonameSearchResult, len(rows))
	for i, row := range rows {
		results[i] = geonameSearchResult{
			GeonameID:   row.Geonameid,
			Name:        row.Name,
			ASCIIName:   row.Asciiname,
			Latitude:    row.Latitude,
			Longitude:   row.Longitude,
			CountryCode: row.CountryCode,
			FeatureCode: row.FeatureCode,
			Population:  row.Population,
			Admin1Name:  row.Admin1Name,
			Admin2Name:  row.Admin2Name,
		}
	}

	writeJSON(w, http.StatusOK, geonameSearchResponse{Results: results})
}
