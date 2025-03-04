-- name: CreateSession :one
INSERT INTO sessions (
  id, created_at, expires_at, secret_hash, data
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

-- name: SetSessionData :exec
UPDATE sessions
SET data = ?
WHERE id = ?;

-- name: SetSessionUserID :exec
UPDATE sessions
SET user_id = ?
WHERE id = ?;
