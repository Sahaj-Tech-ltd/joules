-- name: CreateMeal :one
INSERT INTO meals (user_id, timestamp, meal_type, photo_path, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetMealsByDate :many
SELECT * FROM meals
WHERE user_id = $1 AND timestamp::date = $2
ORDER BY timestamp;

-- name: GetRecentMeals :many
SELECT m.* FROM meals m
WHERE m.user_id = $1
ORDER BY m.created_at DESC
LIMIT 20;

-- name: DeleteMeal :exec
DELETE FROM meals WHERE id = $1 AND user_id = $2;

-- name: GetMealByID :one
SELECT * FROM meals WHERE id = $1 AND user_id = $2 LIMIT 1;
