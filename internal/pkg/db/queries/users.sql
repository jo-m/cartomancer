-- name: GetUser :one
SELECT * FROM users
WHERE uuid = ? LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = ? LIMIT 1;

-- name: GetUsers :many
SELECT * FROM users ORDER BY uuid;

-- name: CreateUser :one
INSERT INTO users (
  uuid, created_at, updated_at, email, name, password_hash, admin
) VALUES (
  ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: UpdateUser :execrows
UPDATE users
SET updated_at = ?, email = ?, name = ?, admin = ?
WHERE uuid = ?;

-- name: UpdateUserPassword :execrows
UPDATE users
SET updated_at = ?, password_hash = ?
WHERE uuid = ?;

-- name: UpdateUserLastLogin :execrows
UPDATE users
SET last_login_at = ?, last_active_at = ?
WHERE uuid = ?;

-- name: UpdateUserLastActive :execrows
UPDATE users
SET last_active_at = ?
WHERE uuid = ?;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE uuid = ?;
