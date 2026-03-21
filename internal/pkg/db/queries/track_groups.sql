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
  AND total_distance_m <= ?
ORDER BY uuid;

-- name: GetSimilarTracks :many
SELECT t.uuid, t.name, t.total_distance_m
FROM track_group_members tgm1
JOIN track_group_members tgm2 ON tgm2.group_id = tgm1.group_id AND tgm2.track_id != tgm1.track_id
JOIN tracks t ON t.uuid = tgm2.track_id
WHERE tgm1.track_id = ?
ORDER BY t.name;

-- name: ListTrackGroupsWithCountByUser :many
SELECT tg.uuid, COUNT(tgm.track_id) AS member_count,
       CAST(MIN(t.name) AS TEXT) AS sample_name
FROM track_groups tg
JOIN track_group_members tgm ON tgm.group_id = tg.uuid
JOIN tracks t ON t.uuid = tgm.track_id
WHERE tg.user_id = ?
GROUP BY tg.uuid
HAVING COUNT(tgm.track_id) > 1
ORDER BY sample_name;

-- name: GetTrackGroupByUUID :one
SELECT tg.uuid, tg.user_id
FROM track_groups tg
WHERE tg.uuid = ?;

-- name: ListTrackGroupMemberUUIDs :many
SELECT tgm.track_id
FROM track_group_members tgm
WHERE tgm.group_id = ?;
