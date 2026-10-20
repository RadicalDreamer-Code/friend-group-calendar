-- name: SearchUsers :many
-- Users whose username starts with the pattern (a LIKE pattern ending in %),
-- each with the viewer's friendship to them: none, friends, outgoing (the
-- viewer asked) or incoming. Exact match first, then shorter usernames.
SELECT u.id, u.username::text AS username, u.avatar_key,
       (CASE
           WHEN f.status = 'accepted' THEN 'friends'
           WHEN f.requester_id = sqlc.arg(viewer_id) THEN 'outgoing'
           WHEN f.requester_id IS NOT NULL THEN 'incoming'
           ELSE 'none'
       END)::text AS friendship
FROM users u
LEFT JOIN friendships f
    ON (f.requester_id = u.id AND f.addressee_id = sqlc.arg(viewer_id))
    OR (f.requester_id = sqlc.arg(viewer_id) AND f.addressee_id = u.id)
WHERE u.username::text LIKE sqlc.arg(pattern)::text
  AND u.id <> sqlc.arg(viewer_id)
ORDER BY u.username::text = sqlc.arg(exact)::text DESC, length(u.username::text), u.username::text
LIMIT sqlc.arg(max_results);

-- name: ListFriends :many
SELECT u.id, u.username::text AS username, u.avatar_key, f.accepted_at
FROM friendships f
JOIN users u ON u.id = CASE WHEN f.requester_id = sqlc.arg(user_id) THEN f.addressee_id ELSE f.requester_id END
WHERE (f.requester_id = sqlc.arg(user_id) OR f.addressee_id = sqlc.arg(user_id))
  AND f.status = 'accepted'
ORDER BY u.username::text;

-- name: ListFriendRequests :many
-- Pending requests the user sent or received, newest first.
SELECT u.id, u.username::text AS username, u.avatar_key, f.created_at,
       (f.addressee_id = sqlc.arg(user_id))::boolean AS incoming
FROM friendships f
JOIN users u ON u.id = CASE WHEN f.requester_id = sqlc.arg(user_id) THEN f.addressee_id ELSE f.requester_id END
WHERE (f.requester_id = sqlc.arg(user_id) OR f.addressee_id = sqlc.arg(user_id))
  AND f.status = 'pending'
ORDER BY f.created_at DESC;

-- name: GetFriendship :one
SELECT * FROM friendships
WHERE (requester_id = sqlc.arg(user_a) AND addressee_id = sqlc.arg(user_b))
   OR (requester_id = sqlc.arg(user_b) AND addressee_id = sqlc.arg(user_a));

-- name: CreateFriendRequest :execrows
-- Does nothing when the pair already has a row, in either direction.
INSERT INTO friendships (requester_id, addressee_id)
VALUES (sqlc.arg(requester_id), sqlc.arg(addressee_id))
ON CONFLICT DO NOTHING;

-- name: AcceptFriendRequest :execrows
UPDATE friendships SET status = 'accepted', accepted_at = now()
WHERE requester_id = sqlc.arg(requester_id) AND addressee_id = sqlc.arg(addressee_id) AND status = 'pending';

-- name: DeleteFriendship :execrows
-- Deletes the pair's row in either direction if it has the given status.
DELETE FROM friendships
WHERE ((requester_id = sqlc.arg(user_a) AND addressee_id = sqlc.arg(user_b))
    OR (requester_id = sqlc.arg(user_b) AND addressee_id = sqlc.arg(user_a)))
  AND status = sqlc.arg(status);
