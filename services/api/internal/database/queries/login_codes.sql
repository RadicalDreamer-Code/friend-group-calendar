-- name: CreateLoginCode :one
INSERT INTO login_codes (email, code_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- Counts one attempt and returns the hash to compare against. No row means
-- there is no usable code: unknown challenge, another address, expired, used,
-- or out of attempts.
-- name: AttemptLoginCode :one
UPDATE login_codes
SET attempts = attempts + 1
WHERE id = sqlc.arg(id)
  AND email = sqlc.arg(email)
  AND expires_at > sqlc.arg(now)
  AND attempts < sqlc.arg(max_attempts)
RETURNING code_hash;

-- Deletes the code that was checked, so it can only be used once even when
-- two requests verify it at the same time.
-- name: ConsumeLoginCode :execrows
DELETE FROM login_codes WHERE id = $1;

-- name: DeleteExpiredLoginCodes :execrows
DELETE FROM login_codes WHERE expires_at < $1;
