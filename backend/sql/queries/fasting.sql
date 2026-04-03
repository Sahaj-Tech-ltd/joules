-- name: StartFast :exec
UPDATE user_goals SET current_fast_start = NOW(), updated_at = NOW() WHERE user_id = $1;

-- name: BreakFast :exec
UPDATE user_goals SET current_fast_start = NULL, fasting_streak = $2, updated_at = NOW() WHERE user_id = $1;

-- name: UpdateEatingWindow :exec
UPDATE user_goals SET eating_window_start = $2, updated_at = NOW() WHERE user_id = $1;

-- name: GetFastingStatus :one
SELECT eating_window_start, current_fast_start, fasting_streak
FROM user_goals WHERE user_id = $1;
