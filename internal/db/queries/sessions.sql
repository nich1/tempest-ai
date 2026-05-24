-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT
    sessions.id,
    sessions.user_id,
    sessions.token_hash,
    sessions.expires_at,
    sessions.created_at,
    sessions.last_seen_at,
    users.email AS user_email,
    users.created_at AS user_created_at
FROM sessions
JOIN users ON users.id = sessions.user_id
WHERE sessions.token_hash = $1
  AND sessions.expires_at > now()
LIMIT 1;

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = now()
WHERE token_hash = $1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteSessionsForUser :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= now();
