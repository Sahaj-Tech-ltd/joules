-- name: CreateFoodItem :one
INSERT INTO food_items (meal_id, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetFoodItemsByMeal :many
SELECT * FROM food_items WHERE meal_id = $1 ORDER BY id;

-- name: UpdateFoodItem :exec
UPDATE food_items
SET name = $1, calories = $2, protein_g = $3, carbs_g = $4, fat_g = $5, fiber_g = $6, serving_size = $7
WHERE food_items.id = $8 AND food_items.meal_id IN (SELECT meals.id FROM meals WHERE meals.user_id = $9);

-- name: DeleteFoodItemByUser :exec
DELETE FROM food_items WHERE food_items.id = @id AND food_items.meal_id IN (SELECT meals.id FROM meals WHERE meals.user_id = @user_id);
