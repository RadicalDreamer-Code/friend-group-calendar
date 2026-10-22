-- name: CreateEntry :one
INSERT INTO entries (calendar_id, title, day, color, created_by)
VALUES (sqlc.arg(calendar_id), sqlc.arg(title), sqlc.arg(day), sqlc.arg(color), sqlc.arg(created_by))
RETURNING *;

-- name: GetEntry :one
SELECT * FROM entries WHERE id = sqlc.arg(id) AND calendar_id = sqlc.arg(calendar_id);

-- name: GetEntryWithCreator :one
SELECT e.id, e.calendar_id, e.title, e.day, e.color, e.created_by, e.created_at, e.updated_at,
       COALESCE(u.username::text, '')::text AS creator_username, u.avatar_key AS creator_avatar_key
FROM entries e
LEFT JOIN users u ON u.id = e.created_by
WHERE e.id = sqlc.arg(id) AND e.calendar_id = sqlc.arg(calendar_id);

-- name: ListEntriesBetween :many
-- Entries from `from` up to, not including, `until`, by day and then in the order they were added.
SELECT e.id, e.calendar_id, e.title, e.day, e.color, e.created_by, e.created_at, e.updated_at,
       COALESCE(u.username::text, '')::text AS creator_username, u.avatar_key AS creator_avatar_key
FROM entries e
LEFT JOIN users u ON u.id = e.created_by
WHERE e.calendar_id = sqlc.arg(calendar_id) AND e.day >= sqlc.arg(from_day) AND e.day < sqlc.arg(until_day)
ORDER BY e.day, e.created_at, e.id;

-- name: UpdateEntry :one
UPDATE entries
SET title = sqlc.arg(title), day = sqlc.arg(day), color = sqlc.arg(color), updated_at = now()
WHERE id = sqlc.arg(id) AND calendar_id = sqlc.arg(calendar_id)
RETURNING *;

-- name: DeleteEntry :execrows
DELETE FROM entries WHERE id = sqlc.arg(id) AND calendar_id = sqlc.arg(calendar_id);

-- name: CountEntries :one
SELECT count(*)::int FROM entries WHERE calendar_id = $1;

-- name: CountEntriesOnDay :one
SELECT count(*)::int FROM entries WHERE calendar_id = sqlc.arg(calendar_id) AND day = sqlc.arg(day);

-- name: ListEntryResponses :many
-- Answers of the people who are still members, oldest first.
SELECT r.entry_id, r.user_id, r.status
FROM entry_responses r
JOIN entries e ON e.id = r.entry_id
JOIN calendar_members m ON m.calendar_id = e.calendar_id AND m.user_id = r.user_id
WHERE r.entry_id = ANY (sqlc.arg(entry_ids)::uuid[])
ORDER BY r.responded_at, r.user_id;

-- name: SetEntryResponse :exec
INSERT INTO entry_responses (entry_id, user_id, status)
VALUES (sqlc.arg(entry_id), sqlc.arg(user_id), sqlc.arg(status))
ON CONFLICT (entry_id, user_id) DO UPDATE
SET status = excluded.status, responded_at = now()
WHERE entry_responses.status <> excluded.status;

-- name: DeleteEntryResponse :exec
DELETE FROM entry_responses WHERE entry_id = sqlc.arg(entry_id) AND user_id = sqlc.arg(user_id);

-- name: DeleteEntryResponses :exec
-- Everyone's answers to an entry, for when its date moves.
DELETE FROM entry_responses WHERE entry_id = $1;

-- name: DeleteMemberEntryResponses :exec
-- A member's answers in one calendar, for when they leave or are removed.
DELETE FROM entry_responses r
USING entries e
WHERE e.id = r.entry_id AND e.calendar_id = sqlc.arg(calendar_id) AND r.user_id = sqlc.arg(user_id);
