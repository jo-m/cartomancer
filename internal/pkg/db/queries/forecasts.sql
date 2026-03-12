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

-- name: DeleteOutdatedForecastFiles :execrows
-- Deletes all forecast_files rows whose valid_time is before the given cutoff.
-- The forecast_files_delete_blob trigger cascades the deletion to orphaned blobs.
DELETE FROM forecast_files
WHERE valid_time < ?;
