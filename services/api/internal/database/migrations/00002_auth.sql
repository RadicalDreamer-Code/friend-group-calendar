-- +goose Up
CREATE TABLE users (
    id           uuid PRIMARY KEY DEFAULT uuidv7(),
    email        citext NOT NULL UNIQUE,
    -- NULL until the user picks one right after their first sign-in.
    -- The ::text cast keeps the check case-sensitive (citext's ~ is not).
    username     citext UNIQUE CHECK (username::text ~ '^[a-z0-9_.]{3,20}$'),
    -- Object key in the avatar bucket; the API serves it under /avatars/<key>.
    avatar_key   text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now()
);

-- One pending code per email: requesting a new code replaces the old one.
-- Keyed by email, not user, because the account is created on the first verify.
CREATE TABLE login_codes (
    email      citext PRIMARY KEY,
    code_hash  bytea NOT NULL,
    attempts   integer NOT NULL DEFAULT 0,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id           uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- SHA-256 of the session token; the token itself is never stored.
    token_hash   bytea NOT NULL UNIQUE,
    device_name  text NOT NULL,
    platform     text NOT NULL CHECK (platform IN ('web', 'ios', 'android')),
    user_agent   text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- +goose Down
DROP TABLE sessions;
DROP TABLE login_codes;
DROP TABLE users;
