-- name: CreateForecastFile :one
INSERT INTO forecast_files (
    created_at, reference_time, valid_time, variable,
    bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
    blob_id
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?
)
RETURNING *;

-- name: GetLatestForecastReferenceTime :one
SELECT reference_time FROM forecast_files
ORDER BY reference_time DESC
LIMIT 1;

-- name: ListForecastFileKeysForReferenceTime :many
-- Returns (variable, valid_time) pairs for all stored files with the given reference time.
SELECT variable, valid_time FROM forecast_files
WHERE reference_time = ?;

-- name: DeleteOutdatedForecastFiles :execrows
-- Deletes all forecast_files rows whose valid_time is before the given cutoff.
-- The forecast_files_delete_blob trigger cascades the deletion to orphaned blobs.
DELETE FROM forecast_files
WHERE valid_time < ?;

-- name: ListForecastFilesForWindow :many
-- Returns forecast_files rows (with blob content) for the latest reference_time
-- whose valid_time falls within [start, end] and whose bounding box overlaps
-- the query box.
SELECT mf.*, b.content, b.compression FROM forecast_files mf
JOIN blobs b ON b.id = mf.blob_id
WHERE mf.reference_time = (SELECT MAX(reference_time) FROM forecast_files)
  AND mf.valid_time >= sqlc.arg(start)
  AND mf.valid_time <= sqlc.arg(end)
  AND (mf.bounds_min_lat IS NULL OR mf.bounds_min_lat <= sqlc.arg(max_lat))
  AND (mf.bounds_max_lat IS NULL OR mf.bounds_max_lat >= sqlc.arg(min_lat))
  AND (mf.bounds_min_lon IS NULL OR mf.bounds_min_lon <= sqlc.arg(max_lon))
  AND (mf.bounds_max_lon IS NULL OR mf.bounds_max_lon >= sqlc.arg(min_lon))
ORDER BY mf.variable, mf.valid_time;

-- name: CreateForecastGrid :one
-- Inserts a new grid constants row for the given reference time.
INSERT INTO forecast_grid (
    created_at, reference_time, blob_id
) VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: GetLatestForecastGrid :one
-- Returns the newest grid constants row by reference_time.
SELECT mg.*, b.content, b.compression FROM forecast_grid mg
JOIN blobs b ON b.id = mg.blob_id
ORDER BY mg.reference_time DESC
LIMIT 1;

-- name: ForecastGridExistsForReferenceTime :one
-- Checks whether a grid constants row exists for the given reference time.
SELECT EXISTS(
    SELECT 1 FROM forecast_grid WHERE reference_time = ?
) AS ok;
