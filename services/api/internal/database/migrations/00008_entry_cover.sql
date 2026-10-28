-- +goose Up
-- An optional cover scene for an entry. The client draws the scenes, so
-- there is nothing to upload; NULL means no cover.
ALTER TABLE entries ADD COLUMN cover text
    CONSTRAINT entries_cover_check CHECK (cover IN ('lake', 'mountains', 'beach', 'city', 'night_out', 'dinner', 'game_night'));

-- +goose Down
ALTER TABLE entries DROP COLUMN cover;
