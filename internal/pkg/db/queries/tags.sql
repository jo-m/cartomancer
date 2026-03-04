-- name: UpsertTag :one
INSERT INTO tags (tag, user_id) VALUES (?, ?)
ON CONFLICT (tag, user_id) DO UPDATE SET tag = tag
RETURNING *;

-- name: GetTagsByTrackID :many
SELECT t.tag FROM tags t
JOIN track_tags tt ON tt.tag_id = t.id
WHERE tt.track_id = ?
ORDER BY t.tag;

-- name: DeleteTrackTags :exec
DELETE FROM track_tags WHERE track_id = ?;

-- name: CreateTrackTag :exec
INSERT INTO track_tags (track_id, tag_id) VALUES (?, ?);

-- name: SuggestTags :many
SELECT tag FROM tags WHERE user_id = ? AND tag LIKE ? ORDER BY tag LIMIT 5;
