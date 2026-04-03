-- name: CreateGoals :one
INSERT INTO user_goals (user_id, objective, diet_plan, fasting_window, daily_calorie_target, daily_protein_g, daily_carbs_g, daily_fat_g)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (user_id) DO UPDATE SET
    objective = $2, diet_plan = $3, fasting_window = $4,
    daily_calorie_target = $5, daily_protein_g = $6, daily_carbs_g = $7, daily_fat_g = $8,
    updated_at = NOW()
RETURNING *;

-- name: GetGoals :one
SELECT * FROM user_goals WHERE user_id = $1 LIMIT 1;
