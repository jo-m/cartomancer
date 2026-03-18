-- name: DeleteTrackGroupsByUser :exec
DELETE FROM track_groups WHERE user_id = ?;

-- name: CreateTrackGroup :exec
INSERT INTO track_groups (uuid, created_at, user_id) VALUES (?, ?, ?);

-- name: CreateTrackGroupMember :exec
INSERT INTO track_group_members (group_id, track_id) VALUES (?, ?);

-- name: ListTrackGroupsByUser :many
SELECT tg.uuid, tg.created_at, tg.user_id, tgm.track_id
FROM track_groups tg
JOIN track_group_members tgm ON tgm.group_id = tg.uuid
WHERE tg.user_id = ?
ORDER BY tg.uuid, tgm.track_id;

-- name: ListGroupableTracksByUser :many
SELECT uuid, blob_id, file_format, original_filename, total_distance_m, track_type
FROM tracks
WHERE user_id = ?
  AND initial_editing_completed = 1
  AND total_distance_m <= ?
ORDER BY uuid;

-- name: GetTrackGroupState :one
SELECT latest_track_uuid, created_at FROM track_group_state WHERE user_id = ?;

-- name: UpsertTrackGroupState :exec
INSERT INTO track_group_state (user_id, latest_track_uuid, created_at)
VALUES (?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET latest_track_uuid = excluded.latest_track_uuid, created_at = excluded.created_at;

-- name: DeleteTrackGroupState :exec
DELETE FROM track_group_state WHERE user_id = ?;

-- name: GetLatestTrackUUIDByUser :one
SELECT uuid FROM tracks
WHERE user_id = ? AND initial_editing_completed = 1
ORDER BY uuid DESC LIMIT 1;

-- name: GetSimilarTracks :many
SELECT t.uuid, t.name, t.total_distance_m
FROM track_group_members tgm1
JOIN track_group_members tgm2 ON tgm2.group_id = tgm1.group_id AND tgm2.track_id != tgm1.track_id
JOIN tracks t ON t.uuid = tgm2.track_id
WHERE tgm1.track_id = ?
ORDER BY t.name;
