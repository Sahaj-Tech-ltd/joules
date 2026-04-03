-- name: AddFoodFavorite :one
INSERT INTO food_favorites (user_id, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, lower(name)) DO UPDATE SET
    calories = EXCLUDED.calories,
    protein_g = EXCLUDED.protein_g,
    carbs_g = EXCLUDED.carbs_g,
    fat_g = EXCLUDED.fat_g,
    fiber_g = EXCLUDED.fiber_g,
    serving_size = EXCLUDED.serving_size,
    source = EXCLUDED.source,
    updated_at = NOW()
RETURNING *;

-- name: RemoveFoodFavorite :exec
DELETE FROM food_favorites WHERE id = $1 AND user_id = $2;

-- name: GetFoodFavorites :many
SELECT * FROM food_favorites
WHERE user_id = $1
ORDER BY use_count DESC, updated_at DESC;

-- name: IncrementFavoriteUseCount :exec
UPDATE food_favorites SET use_count = use_count + 1, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: GetTopFavorites :many
SELECT * FROM food_favorites
WHERE user_id = $1
ORDER BY use_count DESC
LIMIT $2;

-- name: IsFoodFavorite :one
SELECT EXISTS(SELECT 1 FROM food_favorites WHERE user_id = $1 AND lower(name) = lower($2));
