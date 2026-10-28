-- name: CreateEntry :one
INSERT INTO entries (calendar_id, title, day, end_day, color, created_by)
VALUES (sqlc.arg(calendar_id), sqlc.arg(title), sqlc.arg(day), sqlc.arg(end_day), sqlc.arg(color), sqlc.arg(created_by))
RETURNING *;

-- name: GetEntry :one
SELECT * FROM entries WHERE id = sqlc.arg(id) AND calendar_id = sqlc.arg(calendar_id);

-- name: GetEntryWithCreator :one
SELECT e.id, e.calendar_id, e.title, e.day, e.end_day, e.color, e.created_by, e.created_at, e.updated_at,
       COALESCE(u.username::text, '')::text AS creator_username, u.avatar_key AS creator_avatar_key
FROM entries e
LEFT JOIN users u ON u.id = e.created_by
WHERE e.id = sqlc.arg(id) AND e.calendar_id = sqlc.arg(calendar_id);

-- name: ListEntriesBetween :many
-- Entries that overlap the days from `from` up to, not including, `until`,
-- by first day and then in the order they were added.
SELECT e.id, e.calendar_id, e.title, e.day, e.end_day, e.color, e.created_by, e.created_at, e.updated_at,
       COALESCE(u.username::text, '')::text AS creator_username, u.avatar_key AS creator_avatar_key
FROM entries e
LEFT JOIN users u ON u.id = e.created_by
WHERE e.calendar_id = sqlc.arg(calendar_id) AND e.day < sqlc.arg(until_day) AND e.end_day >= sqlc.arg(from_day)
ORDER BY e.day, e.created_at, e.id;

-- name: UpdateEntry :one
UPDATE entries
SET title = sqlc.arg(title), day = sqlc.arg(day), end_day = sqlc.arg(end_day), color = sqlc.arg(color), updated_at = now()
WHERE id = sqlc.arg(id) AND calendar_id = sqlc.arg(calendar_id)
RETURNING *;

-- name: DeleteEntry :execrows
DELETE FROM entries WHERE id = sqlc.arg(id) AND calendar_id = sqlc.arg(calendar_id);

-- name: CountEntries :one
SELECT count(*)::int FROM entries WHERE calendar_id = $1;

-- name: MostEntriesOnADay :one
-- The most entries on any one day from `first_day` to `last_day`, leaving out
-- the entry `except_id`.
SELECT COALESCE(max(n), 0)::int FROM (
    SELECT count(*) AS n
    FROM generate_series(sqlc.arg(first_day)::date, sqlc.arg(last_day)::date, interval '1 day') AS d (day)
    JOIN entries e ON e.calendar_id = sqlc.arg(calendar_id) AND e.day <= d.day::date AND e.end_day >= d.day::date
    WHERE e.id <> sqlc.arg(except_id)
    GROUP BY d.day
) per_day;

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
-- Everyone's answers to an entry, for when its dates change.
DELETE FROM entry_responses WHERE entry_id = $1;

-- name: DeleteMemberEntryResponses :exec
-- A member's answers in one calendar, for when they leave or are removed.
DELETE FROM entry_responses r
USING entries e
WHERE e.id = r.entry_id AND e.calendar_id = sqlc.arg(calendar_id) AND r.user_id = sqlc.arg(user_id);

-- name: NextEntry :one
-- The first entry that has not ended before `from_day`, so one that is still
-- running comes first; by first day and then in the order they were added.
SELECT e.id, e.calendar_id, e.title, e.day, e.end_day, e.color, e.created_by, e.created_at, e.updated_at,
       COALESCE(u.username::text, '')::text AS creator_username, u.avatar_key AS creator_avatar_key
FROM entries e
LEFT JOIN users u ON u.id = e.created_by
WHERE e.calendar_id = sqlc.arg(calendar_id) AND e.end_day >= sqlc.arg(from_day)
ORDER BY e.day, e.created_at, e.id
LIMIT 1;
