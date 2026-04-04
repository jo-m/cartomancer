-- name: CreateEmailVerification :one
INSERT INTO email_verifications (
  uuid, created_at, expires_at, user_id, email
) VALUES (
  ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetEmailVerification :one
SELECT * FROM email_verifications
WHERE uuid = ? LIMIT 1;

-- name: GetEmailVerificationByUserID :one
SELECT * FROM email_verifications
WHERE user_id = ? LIMIT 1;

-- name: DeleteEmailVerification :execrows
DELETE FROM email_verifications
WHERE uuid = ?;

-- name: DeleteEmailVerificationsForUser :execrows
DELETE FROM email_verifications
WHERE user_id = ?;

-- name: GetUserIDsWithPendingEmailVerification :many
SELECT DISTINCT user_id FROM email_verifications;

-- name: DeleteExpiredEmailVerifications :execrows
DELETE FROM email_verifications
WHERE datetime(expires_at) < datetime(?);
