-- name: CreateUserSession :one
INSERT INTO user_sessions (
  created_at, secret, user_id
) VALUES (
  ?, ?, ?
)
RETURNING *;

-- name: GetUserSession :one
SELECT * FROM user_sessions
WHERE id = ? LIMIT 1;

-- name: DeleteUserSession :exec
DELETE FROM user_sessions
WHERE id = ?;

-- TODO: clean up sessions periodically
