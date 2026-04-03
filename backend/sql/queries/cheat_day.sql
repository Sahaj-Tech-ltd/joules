-- name: MarkCheatDay :exec
INSERT INTO cheat_days (user_id, date) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnmarkCheatDay :exec
DELETE FROM cheat_days WHERE user_id = $1 AND date = $2;

-- name: IsCheatDay :one
SELECT EXISTS(SELECT 1 FROM cheat_days WHERE user_id = $1 AND date = $2) AS is_cheat_day;

-- name: GetCheatDaysInRange :many
SELECT date FROM cheat_days WHERE user_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date;
