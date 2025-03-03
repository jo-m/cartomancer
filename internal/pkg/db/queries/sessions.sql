-- name: CreateSession :one
INSERT INTO sessions (
  created_at, expires_at, secret_hash, data
) VALUES (
  ?, ?, ?, ?
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

-- TODO: clean up sessions periodically
