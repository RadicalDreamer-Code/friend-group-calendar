-- +goose Up
-- One row per pair of users: a pending request from requester to addressee,
-- or an accepted friendship. Declining a request or removing a friend
-- deletes the row.
CREATE TABLE friendships (
    requester_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    addressee_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status       text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    accepted_at  timestamptz,
    PRIMARY KEY (requester_id, addressee_id),
    CHECK (requester_id <> addressee_id),
    CHECK ((status = 'accepted') = (accepted_at IS NOT NULL))
);

-- At most one row per pair, whoever asked first: two people sending each
-- other a request at the same moment cannot end up with two rows.
CREATE UNIQUE INDEX friendships_pair_idx
    ON friendships (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id));

CREATE INDEX friendships_addressee_id_idx ON friendships (addressee_id);

-- Username search matches prefixes: "ad" finds ada.l and adam.
CREATE INDEX users_username_prefix_idx ON users ((username::text) text_pattern_ops);

-- +goose Down
DROP INDEX users_username_prefix_idx;
DROP TABLE friendships;
