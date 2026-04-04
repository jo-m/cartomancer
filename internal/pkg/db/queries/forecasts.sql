-- name: CreateForecastFile :one
INSERT INTO forecast_files (
    valid_time, valid_until_time, variable, file, forecast_id
) VALUES (
    ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetLatestForecastReferenceTime :one
SELECT f.reference_time FROM forecasts f
ORDER BY f.reference_time DESC
LIMIT 1;

-- name: DeleteOutdatedForecastFiles :execrows
-- Deletes all forecast_files rows whose valid_time is before the given cutoff.
DELETE FROM forecast_files
WHERE datetime(valid_time) < datetime(sqlc.arg(cutoff));

-- name: ListForecastFilesForWindow :many
-- Returns forecast_files rows for the requested time window and bbox.
-- For each (variable, valid_time) pair, returns the file from the newest
-- forecast run that has it, falling back to older runs to fill gaps.
SELECT mf.* FROM forecast_files mf
WHERE datetime(mf.valid_until_time) > datetime(sqlc.arg(start))
  AND datetime(mf.valid_time) <= datetime(sqlc.arg(end))
  AND mf.forecast_id = (
    SELECT mf2.forecast_id FROM forecast_files mf2
    JOIN forecasts f2 ON mf2.forecast_id = f2.id
    WHERE mf2.variable = mf.variable
      AND mf2.valid_time = mf.valid_time
      AND (f2.bounds_min_lat IS NULL OR f2.bounds_min_lat <= sqlc.arg(max_lat))
      AND (f2.bounds_max_lat IS NULL OR f2.bounds_max_lat >= sqlc.arg(min_lat))
      AND (f2.bounds_min_lon IS NULL OR f2.bounds_min_lon <= sqlc.arg(max_lon))
      AND (f2.bounds_max_lon IS NULL OR f2.bounds_max_lon >= sqlc.arg(min_lon))
    ORDER BY f2.reference_time DESC
    LIMIT 1
  )
ORDER BY mf.variable, mf.valid_time;

-- name: CreateForecast :one
-- Inserts a new forecast row for the given reference time.
INSERT INTO forecasts (
    created_at, reference_time,
    bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
    horizontal_grid_file, vertical_grid_file,
    attribution, attribution_href
) VALUES (
    ?, ?,
    ?, ?, ?, ?,
    ?, ?,
    ?, ?
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

-- name: GetForecastByReferenceTime :one
-- Returns the forecast row for the given reference time.
SELECT * FROM forecasts WHERE reference_time = ?;

-- name: CountDistinctForecastVariables :one
-- Returns the number of distinct variables stored for the forecast with the given reference time.
SELECT COUNT(DISTINCT variable) as count FROM forecast_files
WHERE forecast_id = (SELECT id FROM forecasts WHERE reference_time = ?);

-- name: ListForecastFileKeys :many
-- Returns the (variable, valid_time) pairs for files in the given forecast.
SELECT variable, valid_time FROM forecast_files WHERE forecast_id = ?
ORDER BY variable, valid_time;

-- name: ListForecastsWithFiles :many
-- Returns all forecasts LEFT JOINed with their files (excluding blobs).
-- Forecasts without files appear with NULL file columns.
SELECT
    f.id AS forecast_id, f.created_at, f.reference_time,
    f.bounds_min_lat, f.bounds_min_lon, f.bounds_max_lat, f.bounds_max_lon,
    f.attribution, f.attribution_href,
    mf.id AS file_id, mf.valid_time, mf.valid_until_time, mf.variable,
    length(mf.file) AS file_size
FROM forecasts f
LEFT JOIN forecast_files mf ON mf.forecast_id = f.id
ORDER BY f.reference_time DESC, mf.variable, mf.valid_time;
