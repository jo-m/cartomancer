-- name: CreateSession :one
INSERT INTO sessions (
  uuid, created_at, last_active_at, user_id
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions
WHERE uuid = ? LIMIT 1;

-- name: DeleteSession :execrows
DELETE FROM sessions
WHERE uuid = ?;

-- name: UpdateSessionLastActive :execrows
UPDATE sessions
SET last_active_at = ?
WHERE uuid = ?;

-- name: UpdateSessionData :one
INSERT INTO sessions_data (
  session_id, key, data
) VALUES (
  ?, ?, ?
)
ON CONFLICT(session_id, key)
DO UPDATE SET data = excluded.data
RETURNING *;

-- name: GetSessionsCount :one
SELECT COUNT(*) FROM sessions;

-- name: DeleteSessionData :execrows
DELETE FROM sessions_data
WHERE session_id = ? AND key = ?;

-- name: GetSessionData :many
SELECT key, data FROM sessions_data WHERE session_id = ? ORDER BY key;

-- name: DeleteOtherUserSessions :execrows
DELETE FROM sessions
WHERE user_id = ? AND uuid != ?;

-- name: DeleteAllUserSessions :execrows
DELETE FROM sessions
WHERE user_id = ?;

-- name: CleanupSessions :execrows
DELETE FROM sessions
WHERE
  datetime(created_at) < datetime(@createdBefore)
  OR datetime(last_active_at) < datetime(@activeBefore);
