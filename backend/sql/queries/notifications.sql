-- name: SavePushSubscription :exec
INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, user_agent)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (endpoint) DO UPDATE SET
    p256dh = EXCLUDED.p256dh,
    auth = EXCLUDED.auth,
    user_agent = EXCLUDED.user_agent;

-- name: DeletePushSubscription :exec
DELETE FROM push_subscriptions WHERE endpoint = $1 AND user_id = $2;

-- name: GetPushSubscriptionsByUser :many
SELECT * FROM push_subscriptions WHERE user_id = $1;

-- name: DeletePushSubscriptionByEndpoint :exec
DELETE FROM push_subscriptions WHERE endpoint = $1;

-- name: GetAllUsersWithPushSubscriptions :many
SELECT DISTINCT user_id FROM push_subscriptions;

-- name: GetNotificationPrefs :one
SELECT * FROM notification_preferences WHERE user_id = $1;

-- name: UpsertNotificationPrefs :one
INSERT INTO notification_preferences (user_id, water_reminders, water_interval_hours, meal_reminders, if_window_reminders, streak_reminders, quiet_start, quiet_end, ntfy_topic)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO UPDATE SET
    water_reminders = EXCLUDED.water_reminders,
    water_interval_hours = EXCLUDED.water_interval_hours,
    meal_reminders = EXCLUDED.meal_reminders,
    if_window_reminders = EXCLUDED.if_window_reminders,
    streak_reminders = EXCLUDED.streak_reminders,
    quiet_start = EXCLUDED.quiet_start,
    quiet_end = EXCLUDED.quiet_end,
    ntfy_topic = EXCLUDED.ntfy_topic,
    updated_at = NOW()
RETURNING *;
