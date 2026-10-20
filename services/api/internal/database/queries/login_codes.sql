-- name: UpsertLoginCode :exec
INSERT INTO login_codes (email, code_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (email) DO UPDATE
SET code_hash = excluded.code_hash,
    attempts = 0,
    expires_at = excluded.expires_at,
    created_at = excluded.created_at;

-- Counts one attempt and returns the hash to compare against. No row means
-- there is no usable code: never requested, expired, or out of attempts.
-- name: AttemptLoginCode :one
UPDATE login_codes
SET attempts = attempts + 1
WHERE email = sqlc.arg(email)
  AND expires_at > sqlc.arg(now)
  AND attempts < sqlc.arg(max_attempts)
RETURNING code_hash;

-- Deletes the exact code that was checked, so a code can only be used once
-- even if a new one was requested in the meantime.
-- name: ConsumeLoginCode :execrows
DELETE FROM login_codes WHERE email = $1 AND code_hash = $2;

-- name: DeleteExpiredLoginCodes :execrows
DELETE FROM login_codes WHERE expires_at < $1;
