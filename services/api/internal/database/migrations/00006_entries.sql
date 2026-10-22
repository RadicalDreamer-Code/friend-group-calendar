-- +goose Up
-- Whole-day entries in a calendar. Owner and editors add and edit them.
CREATE TABLE entries (
    id          uuid PRIMARY KEY DEFAULT uuidv7(),
    calendar_id uuid NOT NULL REFERENCES calendars (id) ON DELETE CASCADE,
    title       text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 80),
    day         date NOT NULL,
    -- One of the preset entry colors in the client's design tokens.
    color       text NOT NULL CHECK (color IN ('fjord', 'sage', 'clay', 'rose', 'plum', 'sand', 'moss', 'slate')),
    created_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX entries_calendar_day_idx ON entries (calendar_id, day);

-- A member's answer to an entry. No row means they have not answered yet.
CREATE TABLE entry_responses (
    entry_id     uuid NOT NULL REFERENCES entries (id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status       text NOT NULL CHECK (status IN ('accepted', 'declined')),
    responded_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (entry_id, user_id)
);

CREATE INDEX entry_responses_user_id_idx ON entry_responses (user_id);

-- +goose Down
DROP TABLE entry_responses;
DROP TABLE entries;
