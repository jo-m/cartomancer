-- name: DeleteTrack :exec
DELETE FROM tracks WHERE uuid = ?;

-- name: DeleteAllTracks :execrows
DELETE FROM tracks;

-- name: CreateTrack :one
INSERT INTO tracks (
  uuid, created_at, updated_at, user_id, blob_id, file_format, original_filename,
  name, description, source, author, author_link_url,
  track_type, link_url,
  sport, sub_sport,
  total_distance_m, total_ascent_m,
  start_lat, start_lon, end_lat, end_lon,
  bounds_min_lat, bounds_min_lon, bounds_max_lat, bounds_max_lon,
  min_elevation_m, max_elevation_m,
  original_created_at,
  public,
  polyline_dp5m_varint, polyline_dp50m_varint
) VALUES (
  ?, ?, ?, ?, ?, ?, ?,
  ?, ?, ?, ?, ?,
  ?, ?,
  ?, ?,
  ?, ?,
  ?, ?, ?, ?,
  ?, ?, ?, ?,
  ?, ?,
  ?,
  ?,
  ?, ?
)
RETURNING *;

-- name: ListTracksByUser :many
SELECT * FROM tracks WHERE user_id = ? ORDER BY created_at DESC;

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

-- name: SetTrackPreviewPolylines :exec
UPDATE tracks
SET polyline_dp5m_varint = ?,
    polyline_dp50m_varint = ?
WHERE uuid = ?;

-- name: CountTracksMissingPreviewPolylines :one
SELECT COUNT(*) FROM tracks
WHERE polyline_dp5m_varint IS NULL OR polyline_dp50m_varint IS NULL;

-- name: NextTrackMissingPreviewPolylines :one
SELECT uuid FROM tracks
WHERE polyline_dp5m_varint IS NULL OR polyline_dp50m_varint IS NULL
ORDER BY uuid LIMIT 1;
