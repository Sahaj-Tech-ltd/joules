-- name: GetAchievements :many
SELECT * FROM achievements
WHERE user_id = $1
ORDER BY unlocked_at DESC;

-- name: UnlockAchievement :one
INSERT INTO achievements (user_id, type, title, description, category, progress_current, progress_target)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, type) DO UPDATE SET
    category = EXCLUDED.category,
    progress_current = EXCLUDED.progress_current,
    progress_target = EXCLUDED.progress_target
RETURNING *;
