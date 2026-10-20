-- +goose Up
-- Every code request gets its own row, identified by a challenge ID that only
-- the requesting client receives. Someone who knows an address can no longer
-- replace its pending code or use up its attempts with wrong guesses.
-- Pending codes live for minutes, so dropping them here costs nothing.
DROP TABLE login_codes;

CREATE TABLE login_codes (
    id         uuid PRIMARY KEY DEFAULT uuidv7(),
    email      citext NOT NULL,
    code_hash  bytea NOT NULL,
    attempts   integer NOT NULL DEFAULT 0,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX login_codes_expires_at_idx ON login_codes (expires_at);

-- +goose Down
DROP TABLE login_codes;

CREATE TABLE login_codes (
    email      citext PRIMARY KEY,
    code_hash  bytea NOT NULL,
    attempts   integer NOT NULL DEFAULT 0,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
