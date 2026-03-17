-- name: InsertGeoname :exec
INSERT OR REPLACE INTO geonames (
    geonameid, name, asciiname, alternatenames,
    latitude, longitude,
    feature_class, feature_code,
    country_code, cc2,
    admin1_code, admin2_code, admin3_code, admin4_code,
    population, elevation, dem,
    timezone, modification_date
) VALUES (
    ?, ?, ?, ?,
    ?, ?,
    ?, ?,
    ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?
);

-- name: DeleteAllGeonames :execrows
DELETE FROM geonames;

-- name: CreateGeonameImport :one
INSERT INTO geoname_imports (created_at, row_count)
VALUES (?, ?)
RETURNING *;

-- name: GetLatestGeonameImport :one
SELECT * FROM geoname_imports
ORDER BY created_at DESC
LIMIT 1;

-- name: DeleteOldGeonameImports :execrows
-- Keeps only the most recent import record.
DELETE FROM geoname_imports
WHERE id NOT IN (
    SELECT id FROM geoname_imports ORDER BY created_at DESC LIMIT 1
);

-- name: ReverseGeocode :many
-- Finds the nearest populated places to a given lat/lon within a bounding box.
-- Uses feature_class 'P' (populated places) for best reverse geocoding results.
SELECT geonameid, name, asciiname, latitude, longitude,
       feature_class, feature_code, country_code, population, timezone,
       admin1_code,
       CAST((latitude - sqlc.arg(lat)) * (latitude - sqlc.arg(lat)) +
       (longitude - sqlc.arg(lon)) * (longitude - sqlc.arg(lon)) AS REAL) AS dist_sq
FROM geonames
WHERE feature_class = 'P'
  AND latitude >= sqlc.arg(min_lat)
  AND latitude <= sqlc.arg(max_lat)
  AND longitude >= sqlc.arg(min_lon)
  AND longitude <= sqlc.arg(max_lon)
ORDER BY dist_sq
LIMIT sqlc.arg(max_results);

-- name: CountGeonames :one
SELECT COUNT(*) FROM geonames;
