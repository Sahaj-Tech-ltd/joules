-- name: LogExercise :one
INSERT INTO exercises (user_id, timestamp, name, duration_min, calories_burned)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetExercisesByDate :many
SELECT * FROM exercises
WHERE user_id = $1 AND timestamp::date = $2
ORDER BY timestamp;
