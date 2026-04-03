-- name: LogSteps :one
INSERT INTO step_logs (user_id, date, step_count, source)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, date) DO UPDATE
    SET step_count = EXCLUDED.step_count,
        source = EXCLUDED.source
RETURNING user_id, date, step_count, source;

-- name: GetStepsByDate :one
SELECT user_id, date, step_count, source
FROM step_logs
WHERE user_id = $1 AND date = $2;

-- name: GetStepsHistory :many
SELECT user_id, date, step_count, source
FROM step_logs
WHERE user_id = $1 AND date >= $2 AND date <= $3
ORDER BY date ASC;
