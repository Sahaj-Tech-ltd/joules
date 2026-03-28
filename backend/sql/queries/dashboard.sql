-- name: GetDailySummary :one
SELECT
    COALESCE(SUM(fi.calories), 0)::int AS total_calories,
    COALESCE(SUM(fi.protein_g), 0)::float8 AS total_protein,
    COALESCE(SUM(fi.carbs_g), 0)::float8 AS total_carbs,
    COALESCE(SUM(fi.fat_g), 0)::float8 AS total_fat,
    COALESCE(SUM(fi.fiber_g), 0)::float8 AS total_fiber,
    COALESCE(
        (SELECT SUM(calories_burned) FROM exercises e WHERE e.user_id = $1 AND e.timestamp::date = $2), 0
    )::int AS total_burned,
    (
        SELECT COALESCE(SUM(amount_ml), 0)::int FROM water_logs w WHERE w.user_id = $1 AND w.date = $2
    ) AS total_water_ml
FROM meals m
JOIN food_items fi ON fi.meal_id = m.id
WHERE m.user_id = $1 AND m.timestamp::date = $2;
