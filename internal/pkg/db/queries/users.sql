-- name: GetUser :one
SELECT * FROM users
WHERE id = ? LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = ? LIMIT 1;

-- name: GetUsers :many
SELECT * FROM users ORDER BY id;

-- name: CreateUser :one
INSERT INTO users (
  id, created_at, updated_at, email, name, password_hash
) VALUES (
  ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: UpdateUser :execrows
UPDATE users
SET updated_at = ?, email = ?, name = ?
WHERE id = ?;

-- name: UpdateUserPassword :execrows
UPDATE users
SET updated_at = ?, password_hash = ?
WHERE id = ?;

-- name: UpdateUserLastLogin :execrows
UPDATE users
SET last_login_at = ?, last_active_at = ?
WHERE id = ?;

-- name: UpdateUserLastActive :execrows
UPDATE users
SET last_active_at = ?
WHERE id = ?;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE id = ?;
