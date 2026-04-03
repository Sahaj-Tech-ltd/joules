-- name: LogWeight :one
INSERT INTO weight_logs (user_id, date, weight_kg)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, date) DO UPDATE SET weight_kg = $3, created_at = NOW()
RETURNING *;

-- name: GetWeightHistory :many
SELECT * FROM weight_logs
WHERE user_id = $1 AND date BETWEEN $2 AND $3
ORDER BY date;
