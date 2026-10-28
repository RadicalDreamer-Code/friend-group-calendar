-- +goose Up
-- Entries run from day to end_day, both included. Single-day entries end on
-- their own day, and an entry lasts at most 31 days.
ALTER TABLE entries ADD COLUMN end_day date;
UPDATE entries SET end_day = day;
ALTER TABLE entries
    ALTER COLUMN end_day SET NOT NULL,
    ADD CONSTRAINT entries_end_day_check CHECK (end_day >= day AND end_day - day <= 30);

-- Month listings and the next entry look for entries that end on or after a day.
CREATE INDEX entries_calendar_end_day_idx ON entries (calendar_id, end_day);

-- +goose Down
DROP INDEX entries_calendar_end_day_idx;
ALTER TABLE entries DROP COLUMN end_day;
