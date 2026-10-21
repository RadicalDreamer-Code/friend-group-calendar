-- name: CreateCalendar :one
INSERT INTO calendars (name) VALUES (sqlc.arg(name)) RETURNING *;

-- name: GetCalendar :one
SELECT * FROM calendars WHERE id = $1;

-- name: GetCalendarForUpdate :one
-- Locks the calendar so changes to it and its members happen one at a time.
SELECT * FROM calendars WHERE id = $1 FOR UPDATE;

-- name: UpdateCalendar :one
UPDATE calendars
SET name = sqlc.arg(name), members_can_invite = sqlc.arg(members_can_invite), updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteCalendar :execrows
DELETE FROM calendars WHERE id = $1;

-- name: ListCalendarsForUser :many
-- The user's calendars with their own membership and the member count.
SELECT c.id, c.name, c.members_can_invite, c.created_at, m.role, m.can_invite,
       (SELECT count(*) FROM calendar_members cm WHERE cm.calendar_id = c.id)::int AS member_count
FROM calendar_members m
JOIN calendars c ON c.id = m.calendar_id
WHERE m.user_id = sqlc.arg(user_id)
ORDER BY lower(c.name), c.id;

-- name: ListCalendarMemberPreviews :many
-- The first few members of each calendar, owner first, for stacked avatars.
SELECT calendar_id, id, username, avatar_key
FROM (
    SELECT m.calendar_id, u.id, u.username::text AS username, u.avatar_key,
           row_number() OVER (
               PARTITION BY m.calendar_id
               ORDER BY (m.role = 'owner') DESC, m.joined_at, u.id
           ) AS position
    FROM calendar_members m
    JOIN users u ON u.id = m.user_id
    WHERE m.calendar_id = ANY (sqlc.arg(calendar_ids)::uuid[])
) previews
WHERE position <= sqlc.arg(per_calendar)::int
ORDER BY calendar_id, position;

-- name: GetCalendarMember :one
SELECT * FROM calendar_members WHERE calendar_id = sqlc.arg(calendar_id) AND user_id = sqlc.arg(user_id);

-- name: ListCalendarMembers :many
-- Owner first, then editors, then viewers, each by username.
SELECT u.id, u.username::text AS username, u.avatar_key, m.role, m.can_invite, m.joined_at
FROM calendar_members m
JOIN users u ON u.id = m.user_id
WHERE m.calendar_id = sqlc.arg(calendar_id)
ORDER BY CASE m.role WHEN 'owner' THEN 0 WHEN 'editor' THEN 1 ELSE 2 END, u.username::text;

-- name: CountCalendarMembers :one
SELECT count(*)::int FROM calendar_members WHERE calendar_id = $1;

-- name: CountOwnedCalendars :one
SELECT count(*)::int FROM calendar_members WHERE user_id = $1 AND role = 'owner';

-- name: AddCalendarMember :execrows
-- Does nothing when the user is already a member.
INSERT INTO calendar_members (calendar_id, user_id, role, can_invite, invited_by)
VALUES (sqlc.arg(calendar_id), sqlc.arg(user_id), sqlc.arg(role), sqlc.arg(can_invite), sqlc.narg(invited_by))
ON CONFLICT DO NOTHING;

-- name: UpdateCalendarMember :execrows
UPDATE calendar_members SET role = sqlc.arg(role), can_invite = sqlc.arg(can_invite)
WHERE calendar_id = sqlc.arg(calendar_id) AND user_id = sqlc.arg(user_id);

-- name: DeleteCalendarMember :execrows
DELETE FROM calendar_members WHERE calendar_id = sqlc.arg(calendar_id) AND user_id = sqlc.arg(user_id);

-- name: AreFriends :one
SELECT EXISTS (
    SELECT 1 FROM friendships
    WHERE ((requester_id = sqlc.arg(user_a) AND addressee_id = sqlc.arg(user_b))
        OR (requester_id = sqlc.arg(user_b) AND addressee_id = sqlc.arg(user_a)))
      AND status = 'accepted'
);
