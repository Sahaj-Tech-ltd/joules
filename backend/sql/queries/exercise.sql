-- name: LogExercise :one
INSERT INTO exercises (user_id, timestamp, name, duration_min, calories_burned)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetExercisesByDate :many
SELECT * FROM exercises
WHERE user_id = $1 AND (timestamp AT TIME ZONE COALESCE($3::text, 'UTC'))::date = $2
ORDER BY timestamp;
