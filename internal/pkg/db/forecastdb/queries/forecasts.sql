-- name: CreateForecastFileMeta :one
-- Inserts the metadata row for a forecast file. The associated blob must be
-- written separately via CreateForecastFileBlob, ideally in the same tx.
INSERT INTO forecast_files (
    valid_time, valid_until_time, variable, file_size, forecast_id
) VALUES (
    ?, ?, ?, ?, ?
)
RETURNING *;

-- name: CreateForecastFileBlob :exec
-- Inserts the GRIB2 payload for a previously-created forecast_files row.
INSERT INTO forecast_file_blobs (forecast_file_id, file) VALUES (?, ?);

-- name: GetLatestForecastReferenceTime :one
SELECT f.reference_time FROM forecasts f
ORDER BY f.reference_time DESC
LIMIT 1;

-- name: DeleteOutdatedForecastFiles :execrows
-- Deletes all forecast_files rows whose valid_time is before the given cutoff.
DELETE FROM forecast_files
WHERE datetime(valid_time) < datetime(sqlc.arg(cutoff));

-- name: DeleteEmptyForecastsOlderThan :execrows
-- Deletes forecasts that have no associated forecast_files and whose
-- reference_time is before the given cutoff.
DELETE FROM forecasts
WHERE datetime(reference_time) < datetime(sqlc.arg(cutoff))
  AND NOT EXISTS (
    SELECT 1 FROM forecast_files WHERE forecast_id = forecasts.id
  );

-- name: ListForecastFilesForWindow :many
-- Returns forecast_files metadata joined with their blob payload for the
-- requested time window and bbox. For each (variable, valid_time) pair,
-- returns the file from the newest forecast run that has it, falling back
-- to older runs to fill gaps.
SELECT
    mf.id, mf.valid_time, mf.valid_until_time, mf.variable,
    mf.file_size, mf.forecast_id,
    b.file
FROM forecast_files mf
JOIN forecast_file_blobs b ON b.forecast_file_id = mf.id
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

-- name: ListForecastFileIDsForWindow :many
-- Returns only the id, variable, and time metadata of forecast_files for a
-- time/bbox window, without loading the (large) GRIB2 blob.
SELECT mf.id, mf.valid_time, mf.valid_until_time, mf.variable FROM forecast_files mf
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

-- name: GetForecastFileBlob :one
-- Returns the GRIB2 blob for a single forecast file by ID.
SELECT file FROM forecast_file_blobs WHERE forecast_file_id = ?;

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
-- Forecasts without files appear with NULL file columns. Reads only the
-- metadata table, so no blob pages are touched.
SELECT
    f.id AS forecast_id, f.created_at, f.reference_time,
    f.bounds_min_lat, f.bounds_min_lon, f.bounds_max_lat, f.bounds_max_lon,
    f.attribution, f.attribution_href,
    mf.id AS file_id, mf.valid_time, mf.valid_until_time, mf.variable,
    mf.file_size
FROM forecasts f
LEFT JOIN forecast_files mf ON mf.forecast_id = f.id
ORDER BY f.reference_time DESC, mf.variable, mf.valid_time;
