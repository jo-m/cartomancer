-- name: CreateTrackComment :exec
INSERT INTO track_comments (uuid, track_id, user_id, body, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetTrackCommentByUUID :one
SELECT
    tc.uuid,
    tc.track_id,
    tc.user_id,
    tc.body,
    tc.deleted,
    tc.created_at,
    tc.updated_at,
    u.name AS user_name
FROM track_comments tc
JOIN users u ON u.uuid = tc.user_id
WHERE tc.uuid = ?;

-- name: ListTrackComments :many
SELECT
    tc.uuid,
    tc.track_id,
    tc.user_id,
    CAST(CASE WHEN tc.deleted = 1 THEN '' ELSE tc.body END AS TEXT) AS body,
    tc.deleted,
    tc.created_at,
    tc.updated_at,
    u.name AS user_name
FROM track_comments tc
JOIN users u ON u.uuid = tc.user_id
WHERE tc.track_id = ?
ORDER BY tc.created_at ASC;

-- name: UpdateTrackCommentBody :execrows
UPDATE track_comments
SET body = ?, updated_at = ?
WHERE uuid = ?;

-- name: SoftDeleteTrackComment :execrows
UPDATE track_comments
SET deleted = 1, updated_at = ?
WHERE uuid = ?;
