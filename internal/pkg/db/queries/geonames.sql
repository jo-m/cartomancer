-- name: InsertGeoname :exec
INSERT OR REPLACE INTO geonames (
    geonameid, name,
    latitude, longitude,
    feature_class, feature_code,
    country_code, cc2,
    admin1_code, admin2_code, admin3_code, admin4_code
) VALUES (
    ?, ?,
    ?, ?,
    ?, ?,
    ?, ?,
    ?, ?, ?, ?
);

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

-- name: CountGeonames :one
SELECT COUNT(*) FROM geonames;

-- name: InsertGeonameAdmin1 :exec
INSERT OR REPLACE INTO geoname_admin1 (code, name, geonameid)
VALUES (?, ?, ?);

-- name: DeleteAllGeonameAdmin1 :execrows
DELETE FROM geoname_admin1;

-- name: CountGeonameAdmin1 :one
SELECT COUNT(*) FROM geoname_admin1;

-- name: GetGeonameAdmin1 :one
SELECT * FROM geoname_admin1 WHERE code = ?;

-- name: InsertGeonameAdmin2 :exec
INSERT OR REPLACE INTO geoname_admin2 (code, name, geonameid)
VALUES (?, ?, ?);

-- name: DeleteAllGeonameAdmin2 :execrows
DELETE FROM geoname_admin2;

-- name: CountGeonameAdmin2 :one
SELECT COUNT(*) FROM geoname_admin2;

-- name: GetGeonameAdmin2 :one
SELECT * FROM geoname_admin2 WHERE code = ?;

-- name: NearestPlace :one
-- Finds the single nearest populated place to a given lat/lon within a bounding box.
SELECT geonameid, name, country_code, admin1_code,
       CAST((latitude - sqlc.arg(lat)) * (latitude - sqlc.arg(lat)) +
       (longitude - sqlc.arg(lon)) * (longitude - sqlc.arg(lon)) AS REAL) AS dist_sq
FROM geonames
WHERE feature_class = 'P'
  AND latitude >= sqlc.arg(min_lat)
  AND latitude <= sqlc.arg(max_lat)
  AND longitude >= sqlc.arg(min_lon)
  AND longitude <= sqlc.arg(max_lon)
ORDER BY dist_sq
LIMIT 1;

-- name: UpsertTrackGeoname :exec
INSERT INTO track_geonames (track_id, label, created_at)
VALUES (?, ?, ?)
ON CONFLICT(track_id) DO UPDATE SET label = excluded.label, created_at = excluded.created_at;

-- name: GetTrackGeoname :one
SELECT * FROM track_geonames WHERE track_id = ?;
