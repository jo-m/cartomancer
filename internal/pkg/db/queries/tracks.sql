-- name: CreateTrack :one
INSERT INTO tracks (
  uuid, created_at, updated_at, user_id, blob_id, file_format,
  name, description, source, author, author_link_url,
  track_type, link_url,
  sport, sub_sport,
  total_distance_m, total_ascent_m,
  start_lat, start_lon, end_lat, end_lon,
  original_created_at
) VALUES (
  ?, ?, ?, ?, ?, ?,
  ?, ?, ?, ?, ?,
  ?, ?,
  ?, ?,
  ?, ?,
  ?, ?, ?, ?,
  ?
)
RETURNING *;

-- name: GetTrackByUUID :one
SELECT * FROM tracks WHERE uuid = ?;

-- name: UpdateTrack :one
UPDATE tracks
SET updated_at = ?,
    name = ?,
    description = ?,
    source = ?,
    author = ?,
    author_link_url = ?,
    track_type = ?,
    link_url = ?,
    sport = ?,
    sub_sport = ?
WHERE uuid = ?
RETURNING *;
