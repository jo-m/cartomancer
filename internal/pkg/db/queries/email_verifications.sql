-- name: CreateEmailVerification :one
INSERT INTO email_verifications (
  uuid, created_at, expires_at, user_id, email, token
) VALUES (
  ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetEmailVerificationByToken :one
SELECT * FROM email_verifications
WHERE token = ? LIMIT 1;

-- name: GetEmailVerificationByEmail :one
SELECT * FROM email_verifications
WHERE email = ? LIMIT 1;

-- name: DeleteEmailVerification :execrows
DELETE FROM email_verifications
WHERE uuid = ?;

-- name: DeleteEmailVerificationsForUser :execrows
DELETE FROM email_verifications
WHERE user_id = ?;
