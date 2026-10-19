-- +goose Up
-- citext gives case-insensitive emails and usernames (used from milestone 2 on).
CREATE EXTENSION IF NOT EXISTS citext;

-- +goose Down
DROP EXTENSION IF EXISTS citext;
