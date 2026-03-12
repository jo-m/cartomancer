-- name: CreateTrackStar :exec
INSERT INTO track_stars (track_id, user_id, created_at)
VALUES (?, ?, ?)
ON CONFLICT DO NOTHING;

-- name: DeleteTrackStar :execrows
DELETE FROM track_stars WHERE track_id = ? AND user_id = ?;

-- name: IsTrackStarredByUser :one
SELECT COUNT(*) FROM track_stars WHERE track_id = ? AND user_id = ?;
