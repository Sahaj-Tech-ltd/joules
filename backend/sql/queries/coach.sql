-- name: SaveCoachMessage :one
INSERT INTO coach_messages (user_id, role, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetCoachHistory :many
SELECT * FROM coach_messages
WHERE user_id = $1
ORDER BY created_at ASC
LIMIT 100;
