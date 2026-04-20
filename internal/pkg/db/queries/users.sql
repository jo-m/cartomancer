-- name: GetUser :one
SELECT * FROM users
WHERE uuid = ? LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = ? LIMIT 1;

-- name: GetUserByName :one
SELECT * FROM users
WHERE lower(name) = lower(?) LIMIT 1;

-- name: GetUsers :many
SELECT * FROM users ORDER BY uuid;

-- name: ListUserUUIDsAfter :many
SELECT uuid FROM users WHERE uuid > ? ORDER BY uuid LIMIT ?;

-- name: CreateUser :one
INSERT INTO users (
  uuid, created_at, updated_at, email, name, password_hash, admin, email_confirmed
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: UpdateUser :execrows
UPDATE users
SET updated_at = ?, email = ?, name = ?, admin = ?, email_confirmed = 1
WHERE uuid = ?;

-- name: UpdateUserName :execrows
UPDATE users
SET updated_at = ?, name = ?
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

-- name: UpdateUserAvatarSeed :execrows
UPDATE users
SET updated_at = ?, avatar_seed = ?
WHERE uuid = ?;

-- name: CountAdmins :one
SELECT COUNT(*) FROM users WHERE admin = 1;

-- name: ConfirmUserEmail :execrows
UPDATE users
SET email_confirmed = 1, updated_at = ?
WHERE uuid = ?;

-- name: UpdateUserEmail :execrows
UPDATE users
SET email = ?, email_confirmed = 1, updated_at = ?
WHERE uuid = ?;

-- name: UpdateUserLocation :execrows
UPDATE users
SET updated_at = ?, location_name = ?, location_lat = ?, location_lon = ?
WHERE uuid = ?;

-- name: DeleteUnconfirmedUsersWithoutVerification :execrows
DELETE FROM users
WHERE email_confirmed = 0
  AND uuid NOT IN (SELECT user_id FROM email_verifications);

-- name: DeleteUser :execrows
DELETE FROM users
WHERE uuid = ?;
