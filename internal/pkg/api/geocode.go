package api

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"jo-m.ch/go/cartomancer/internal/pkg/db/geonamesdb"
	"jo-m.ch/go/cartomancer/internal/pkg/logg"
)

const (
	geonameSearchMinQuery   = 2
	geonameSearchMaxResults = 5
)

type geonameSearchResult struct {
	ID          int64   `json:"id"`
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

// handleSearchGeocodeName searches populated places by name prefix using FTS5
// and returns results with resolved admin1/admin2 names.
func (sv *server) handleSearchGeocodeName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(q) < geonameSearchMinQuery {
		writeError(w, http.StatusBadRequest, "query must be at least 2 characters")
		return
	}

	ftsQuery := fts5PrefixQuery(q)
	rows, err := sv.gd.QueryRO().SearchGeonames(ctx, geonamesdb.SearchGeonamesParams{
		Query:      ftsQuery,
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
			ID:          row.Geonameid,
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

// fts5PrefixQuery sanitises user input for FTS5 prefix search.
// Each whitespace-separated token becomes a quoted prefix token.
// Example: `New York` -> `"New" "York"*`, `Zürich` -> `"Zürich"*`.
func fts5PrefixQuery(input string) string {
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, tok := range tokens {
		if i > 0 {
			sb.WriteByte(' ')
		}
		// Quote each token to prevent FTS5 syntax injection (e.g. AND, OR, NOT, *).
		escaped := strings.ReplaceAll(tok, `"`, `""`)
		sb.WriteByte('"')
		sb.WriteString(escaped)
		sb.WriteByte('"')
	}
	// Append prefix operator to the last token only.
	sb.WriteByte('*')

	return sb.String()
}
