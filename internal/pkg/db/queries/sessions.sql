-- name: CreateSession :one
INSERT INTO sessions (
  id, created_at, last_active_at, secret_hash, user_id
) VALUES (
  ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetSession :one
SELECT * FROM sessions
WHERE id = ? LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = ?;

-- name: UpdateSessionLastActive :exec
UPDATE sessions
SET last_active_at = ?
WHERE id = ?;

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

-- name: DeleteSessionData :exec
DELETE FROM sessions_data
WHERE session_id = ? AND key = ?;

-- name: GetSessionData :many
SELECT key, data FROM sessions_data WHERE session_id = ? ORDER BY key;

-- name: CleanupSessions :execrows
DELETE FROM sessions
WHERE
  created_at < @createdBefore
  OR last_active_at < @activeBefore;
