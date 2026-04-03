-- name: CreateUser :one
INSERT INTO users (email, password_hash, verification_code, verification_code_expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: VerifyUser :exec
UPDATE users
SET verified = TRUE, verification_code = NULL, verification_code_expires_at = NULL, updated_at = NOW()
WHERE id = $1 AND verification_code = $2;

-- name: UpdatePassword :exec
UPDATE users
SET password_hash = $2, updated_at = NOW()
WHERE id = $1;

-- name: GetVerificationCode :one
SELECT verification_code FROM users WHERE id = $1 LIMIT 1;
