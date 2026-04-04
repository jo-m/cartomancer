-- name: UpsertTrackForecast :exec
INSERT INTO track_forecasts (
    track_uuid, created_at, forecast_reference_time, start_time,
    avg_temperature_c, total_precipitation_mm,
    wind_head_ms, wind_right_ms, wind_tail_ms, wind_left_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(track_uuid) DO UPDATE SET
    created_at = excluded.created_at,
    forecast_reference_time = excluded.forecast_reference_time,
    start_time = excluded.start_time,
    avg_temperature_c = excluded.avg_temperature_c,
    total_precipitation_mm = excluded.total_precipitation_mm,
    wind_head_ms = excluded.wind_head_ms,
    wind_right_ms = excluded.wind_right_ms,
    wind_tail_ms = excluded.wind_tail_ms,
    wind_left_ms = excluded.wind_left_ms;

-- name: GetTrackForecast :one
SELECT * FROM track_forecasts WHERE track_uuid = ?;

-- name: ListTrackUUIDsNeedingForecastBatch :many
SELECT t.uuid
FROM tracks t
LEFT JOIN track_forecasts tf ON t.uuid = tf.track_uuid
WHERE (tf.id IS NULL
   OR datetime(tf.forecast_reference_time) != datetime(sqlc.arg(forecast_reference_time))
   OR datetime(tf.start_time) != datetime(sqlc.arg(start_time)))
  AND t.uuid > sqlc.arg(uuid)
ORDER BY t.uuid ASC
LIMIT sqlc.arg(limit);

-- name: DeleteTrackForecast :exec
DELETE FROM track_forecasts WHERE track_uuid = ?;
