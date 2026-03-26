-- name: GetAchievements :many
SELECT * FROM achievements
WHERE user_id = $1
ORDER BY unlocked_at DESC;

-- name: UnlockAchievement :one
INSERT INTO achievements (user_id, type, title, description)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, type) DO NOTHING
RETURNING *;
