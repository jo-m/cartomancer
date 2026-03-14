-- name: CreateForecastFile :one
INSERT INTO forecast_files (
    valid_time, variable, file, forecast_id
) VALUES (
    ?, ?, ?, ?
)
RETURNING *;

-- name: GetLatestForecastReferenceTime :one
SELECT f.reference_time FROM forecasts f
ORDER BY f.reference_time DESC
LIMIT 1;

-- name: DeleteOutdatedForecastFiles :execrows
-- Deletes all forecast_files rows whose valid_time is before the given cutoff.
DELETE FROM forecast_files
WHERE valid_time < ?;

-- name: ListForecastFilesForWindow :many
-- Returns forecast_files rows for the latest reference_time whose valid_time
-- falls within [start, end] and whose bounding box overlaps the query box.
SELECT mf.* FROM forecast_files mf
JOIN forecasts f ON mf.forecast_id = f.id
WHERE f.reference_time = (SELECT MAX(reference_time) FROM forecasts)
  AND mf.valid_time >= sqlc.arg(start)
  AND mf.valid_time <= sqlc.arg(end)
  AND (f.bounds_min_lat IS NULL OR f.bounds_min_lat <= sqlc.arg(max_lat))
  AND (f.bounds_max_lat IS NULL OR f.bounds_max_lat >= sqlc.arg(min_lat))
  AND (f.bounds_min_lon IS NULL OR f.bounds_min_lon <= sqlc.arg(max_lon))
  AND (f.bounds_max_lon IS NULL OR f.bounds_max_lon >= sqlc.arg(min_lon))
ORDER BY mf.variable, mf.valid_time;

-- name: CreateForecast :one
-- Inserts a new forecast row for the given reference time.
INSERT INTO forecasts (
    created_at, reference_time,
    bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
    grid_file
) VALUES (
    ?, ?,
    ?, ?, ?, ?,
    ?
)
RETURNING *;

-- name: GetLatestForecast :one
-- Returns the newest forecast row by reference_time.
SELECT * FROM forecasts
ORDER BY reference_time DESC
LIMIT 1;

-- name: ForecastExistsForReferenceTime :one
-- Checks whether a forecast row exists for the given reference time.
SELECT EXISTS(
    SELECT 1 FROM forecasts WHERE reference_time = ?
) AS ok;
