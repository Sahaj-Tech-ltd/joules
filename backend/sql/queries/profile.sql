-- name: CreateProfile :one
INSERT INTO user_profiles (user_id, name, age, sex, height_cm, weight_kg, target_weight_kg, activity_level, onboarding_complete)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetProfile :one
SELECT * FROM user_profiles WHERE user_id = $1 LIMIT 1;

-- name: UpdateProfile :exec
UPDATE user_profiles
SET name = $2, age = $3, sex = $4, height_cm = $5, weight_kg = $6, target_weight_kg = $7, activity_level = $8, updated_at = NOW()
WHERE user_id = $1;

-- name: CompleteOnboarding :exec
UPDATE user_profiles SET onboarding_complete = TRUE, updated_at = NOW() WHERE user_id = $1;
