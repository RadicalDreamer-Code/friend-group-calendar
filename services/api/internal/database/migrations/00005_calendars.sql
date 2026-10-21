-- +goose Up
CREATE TABLE calendars (
    id                 uuid PRIMARY KEY DEFAULT uuidv7(),
    name               text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 50),
    -- Lets every member invite friends, on top of the per-member right.
    members_can_invite boolean NOT NULL DEFAULT false,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

-- Roles: owner (exactly one, manages the calendar and its members), editor
-- (adds and edits entries) and viewer (sees entries and answers them).
-- can_invite lets an editor or viewer invite friends; the owner always can.
CREATE TABLE calendar_members (
    calendar_id uuid NOT NULL REFERENCES calendars (id) ON DELETE CASCADE,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role        text NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    can_invite  boolean NOT NULL DEFAULT false,
    invited_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    joined_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (calendar_id, user_id)
);

CREATE UNIQUE INDEX calendar_members_one_owner_idx ON calendar_members (calendar_id) WHERE role = 'owner';
CREATE INDEX calendar_members_user_id_idx ON calendar_members (user_id);

-- +goose Down
DROP TABLE calendar_members;
DROP TABLE calendars;
