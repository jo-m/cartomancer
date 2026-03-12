-- name: CreateForecastFile :one
INSERT INTO forecast_files (
    created_at, reference_time, valid_time, variable, horizon_secs,
    bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
    blob_id
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?
)
RETURNING *;

-- name: GetLatestForecastReferenceTime :one
SELECT reference_time FROM forecast_files
ORDER BY reference_time DESC
LIMIT 1;
