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
-- Returns all of the user's tags matching the given LIKE pattern together with
-- the number of the user's tracks that carry each tag. Tags with no tracks are
-- included with a count of zero. Ordered by track count desc, then tag asc.
-- Only tracks owned by the tag owner are counted; since tracks are always
-- visible to their owner, this naturally respects visibility.
SELECT t.tag, COUNT(tt.track_id) AS n_tracks
FROM tags t
LEFT JOIN track_tags tt ON tt.tag_id = t.id
LEFT JOIN tracks tr ON tr.uuid = tt.track_id AND tr.user_id = t.user_id
WHERE t.user_id = ? AND t.tag LIKE ?
GROUP BY t.id, t.tag
ORDER BY n_tracks DESC, t.tag ASC;
