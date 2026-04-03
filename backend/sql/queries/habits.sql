-- name: GetUserStats :one
SELECT * FROM user_stats WHERE user_id = $1;

-- name: UpsertUserStats :one
INSERT INTO user_stats (user_id, total_points, streak_days, last_active_date, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (user_id) DO UPDATE
    SET total_points = EXCLUDED.total_points,
        streak_days = EXCLUDED.streak_days,
        last_active_date = EXCLUDED.last_active_date,
        updated_at = NOW()
RETURNING *;
