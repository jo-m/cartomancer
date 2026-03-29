package db

import "context"

const searchGeonamesFTS = `
SELECT
    g.geonameid,
    g.name,
    g.asciiname,
    g.latitude,
    g.longitude,
    g.country_code,
    g.feature_code,
    g.population,
    COALESCE(a1.name, '') AS admin1_name,
    COALESCE(a2.name, '') AS admin2_name
FROM geonames g
LEFT JOIN geoname_admin1 a1
    ON a1.code = g.country_code || '.' || g.admin1_code
LEFT JOIN geoname_admin2 a2
    ON a2.code = g.country_code || '.' || g.admin1_code || '.' || g.admin2_code
WHERE g.geonameid IN (SELECT rowid FROM geonames_fts WHERE geonames_fts MATCH ?1)
  AND g.feature_class = 'P'
ORDER BY g.population DESC
LIMIT ?2
`

// SearchGeonamesParams holds parameters for [Queries.SearchGeonames].
type SearchGeonamesParams struct {
	// Query is an FTS5 match expression (e.g. `"zurich"*`).
	Query string
	// MaxResults caps the number of returned rows.
	MaxResults int64
}

// SearchGeonamesRow holds a single result from [Queries.SearchGeonames].
type SearchGeonamesRow struct {
	Geonameid   int64
	Name        string
	Asciiname   string
	Latitude    float64
	Longitude   float64
	CountryCode string
	FeatureCode string
	Population  int64
	Admin1Name  string
	Admin2Name  string
}

// SearchGeonames searches populated places by name prefix via FTS5, returning
// results with admin1/admin2 names joined. Only populated places
// (feature_class = 'P') are returned.
func (q *Queries) SearchGeonames(ctx context.Context, arg SearchGeonamesParams) ([]SearchGeonamesRow, error) {
	rows, err := q.db.QueryContext(ctx, searchGeonamesFTS, arg.Query, arg.MaxResults)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SearchGeonamesRow
	for rows.Next() {
		var i SearchGeonamesRow
		if err := rows.Scan(
			&i.Geonameid,
			&i.Name,
			&i.Asciiname,
			&i.Latitude,
			&i.Longitude,
			&i.CountryCode,
			&i.FeatureCode,
			&i.Population,
			&i.Admin1Name,
			&i.Admin2Name,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return items, rows.Err()
}
