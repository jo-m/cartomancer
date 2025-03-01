-- name: GetUser :one
SELECT * FROM users
WHERE id = ? LIMIT 1;

-- name: GetUserByName :one
SELECT * FROM users
WHERE username = ? LIMIT 1;

-- name: GetUsers :many
SELECT * FROM users ORDER BY id;

-- name: CreateUser :one
INSERT INTO users (
  created_at, updated_at, username, email, password_hash, biography
) VALUES (
  ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: UpdateUser :exec
UPDATE users
SET updated_at = ?, username = ?, email = ?, password_hash = ?, biography = ?
WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;
