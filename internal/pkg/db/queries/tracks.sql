-- name: CreateTrack :one
INSERT INTO tracks (
  uuid, created_at, updated_at, user_id, blob_id, file_format, original_filename,
  name, description, source, author, author_link_url,
  track_type, link_url,
  sport, sub_sport,
  total_distance_m, total_ascent_m,
  start_lat, start_lon, end_lat, end_lon,
  bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
  original_created_at,
  public,
  preview_svg_blob_id
) VALUES (
  ?, ?, ?, ?, ?, ?, ?,
  ?, ?, ?, ?, ?,
  ?, ?,
  ?, ?,
  ?, ?,
  ?, ?, ?, ?,
  ?, ?, ?, ?,
  ?,
  ?,
  ?
)
RETURNING *;

-- name: ListTracksForEditing :many
SELECT * FROM tracks WHERE user_id = ? AND initial_editing_completed = 0 ORDER BY created_at DESC;

-- name: CountTracksByUser :one
SELECT COUNT(*) FROM tracks WHERE user_id = ?;

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
    sub_sport = ?,
    public = ?
WHERE uuid = ?
RETURNING *;
