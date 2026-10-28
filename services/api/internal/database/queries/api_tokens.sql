-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, name, token_hash, created_at, expires_at)
VALUES (sqlc.arg(user_id), sqlc.arg(name), sqlc.arg(token_hash), sqlc.arg(created_at), sqlc.arg(expires_at))
RETURNING *;

-- name: CountLiveAPITokens :one
SELECT count(*)::int FROM api_tokens
WHERE user_id = sqlc.arg(user_id) AND revoked_at IS NULL AND expires_at > sqlc.arg(now);

-- name: ListLiveAPITokens :many
SELECT * FROM api_tokens
WHERE user_id = sqlc.arg(user_id) AND revoked_at IS NULL AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC, id DESC;

-- name: GetLiveAPITokenByHash :one
SELECT sqlc.embed(api_tokens), sqlc.embed(users)
FROM api_tokens
JOIN users ON users.id = api_tokens.user_id
WHERE api_tokens.token_hash = sqlc.arg(token_hash)
  AND api_tokens.revoked_at IS NULL
  AND api_tokens.expires_at > sqlc.arg(now);

-- name: TouchAPIToken :exec
UPDATE api_tokens SET last_used_at = sqlc.arg(now)::timestamptz WHERE id = sqlc.arg(id);

-- name: RevokeAPIToken :execrows
UPDATE api_tokens SET revoked_at = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND revoked_at IS NULL AND expires_at > sqlc.arg(now)::timestamptz;

-- Signing out every other device also ends every access token, so a stolen
-- session cannot leave a token behind.
-- name: RevokeAllAPITokens :execrows
UPDATE api_tokens SET revoked_at = sqlc.arg(now)::timestamptz
WHERE user_id = sqlc.arg(user_id) AND revoked_at IS NULL AND expires_at > sqlc.arg(now)::timestamptz;

-- Like sessions, expired and revoked tokens are kept for a while, then removed.
-- name: DeleteStaleAPITokens :execrows
DELETE FROM api_tokens WHERE expires_at < sqlc.arg(before) OR revoked_at < sqlc.arg(before);
