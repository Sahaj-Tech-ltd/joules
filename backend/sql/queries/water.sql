-- name: LogWater :one
INSERT INTO water_logs (user_id, date, amount_ml)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetWaterByDate :one
SELECT COALESCE(SUM(amount_ml), 0)::int AS total
FROM water_logs
WHERE user_id = $1 AND date = $2;
