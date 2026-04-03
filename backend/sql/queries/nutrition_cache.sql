-- name: GetNutritionCache :one
SELECT * FROM nutrition_cache WHERE lower(query) = lower($1) LIMIT 1;

-- name: UpsertNutritionCache :one
INSERT INTO nutrition_cache (query, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (lower(query)) DO UPDATE SET
    name = EXCLUDED.name,
    calories = EXCLUDED.calories,
    protein_g = EXCLUDED.protein_g,
    carbs_g = EXCLUDED.carbs_g,
    fat_g = EXCLUDED.fat_g,
    fiber_g = EXCLUDED.fiber_g,
    serving_size = EXCLUDED.serving_size,
    source = EXCLUDED.source,
    created_at = NOW()
RETURNING *;
