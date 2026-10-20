-- name: UpsertUserByEmail :one
INSERT INTO users (email)
VALUES ($1)
ON CONFLICT (email) DO UPDATE SET last_seen_at = now()
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserForUpdate :one
SELECT * FROM users WHERE id = $1 FOR UPDATE;

-- name: UsernameTaken :one
SELECT EXISTS (SELECT 1 FROM users WHERE username = sqlc.arg(username)::citext AND id <> sqlc.arg(except_user_id));

-- name: UpdateUsername :one
UPDATE users SET username = sqlc.arg(username)::citext, updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: SetAvatarKey :one
UPDATE users SET avatar_key = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: TouchUserLastSeen :exec
UPDATE users SET last_seen_at = $2 WHERE id = $1;
