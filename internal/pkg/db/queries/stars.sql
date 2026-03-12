-- name: CreateTrackStar :exec
INSERT INTO track_stars (track_id, user_id, created_at)
VALUES (?, ?, ?)
ON CONFLICT DO NOTHING;

-- name: DeleteTrackStar :execrows
DELETE FROM track_stars WHERE track_id = ? AND user_id = ?;

-- name: GetStarredTracksPublicOnly :many
SELECT tracks.* FROM track_stars
JOIN tracks ON tracks.uuid = track_stars.track_id
WHERE track_stars.user_id = ?
  AND tracks.public = 1
ORDER BY track_stars.created_at DESC;

-- name: GetStarredTracksVisible :many
SELECT tracks.* FROM track_stars
JOIN tracks ON tracks.uuid = track_stars.track_id
WHERE track_stars.user_id = ?
  AND (tracks.public = 1 OR tracks.user_id = ?)
ORDER BY track_stars.created_at DESC;
