-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, device_name, platform, user_agent, created_at, last_used_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $6, $7)
RETURNING *;

-- name: GetActiveSessionByTokenHash :one
SELECT sqlc.embed(sessions), sqlc.embed(users)
FROM sessions
JOIN users ON users.id = sessions.user_id
WHERE sessions.token_hash = sqlc.arg(token_hash)
  AND sessions.revoked_at IS NULL
  AND sessions.expires_at > sqlc.arg(now);

-- name: RenewSession :exec
UPDATE sessions SET last_used_at = $2, expires_at = $3 WHERE id = $1;

-- name: ListActiveSessions :many
SELECT * FROM sessions
WHERE user_id = sqlc.arg(user_id)
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)
ORDER BY last_used_at DESC;

-- name: RevokeSession :execrows
UPDATE sessions SET revoked_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND revoked_at IS NULL;

-- name: RevokeOtherSessions :execrows
UPDATE sessions SET revoked_at = sqlc.arg(now)::timestamptz
WHERE user_id = sqlc.arg(user_id) AND id <> sqlc.arg(keep_id) AND revoked_at IS NULL;

-- Expired and revoked sessions are kept for a while so the device list and
-- logs stay explainable, then removed.
-- name: DeleteStaleSessions :execrows
DELETE FROM sessions WHERE expires_at < sqlc.arg(before) OR revoked_at < sqlc.arg(before);
