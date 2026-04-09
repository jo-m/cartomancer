-- name: UpsertTrackGeoname :exec
INSERT INTO track_geonames (track_id, label, created_at)
VALUES (?, ?, ?)
ON CONFLICT(track_id) DO UPDATE SET label = excluded.label, created_at = excluded.created_at;

-- name: GetTrackGeoname :one
SELECT * FROM track_geonames WHERE track_id = ?;
