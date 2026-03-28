// Code generated manually — matches sqlc output conventions.

package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type PushSubscription struct {
	ID        string
	UserID    string
	Endpoint  string
	P256dh    string
	Auth      string
	UserAgent string
	CreatedAt pgtype.Timestamptz
}

type NotificationPreferences struct {
	UserID              string
	WaterReminders      bool
	WaterIntervalHours  int32
	MealReminders       bool
	IfWindowReminders   bool
	StreakReminders     bool
	QuietStart          int32
	QuietEnd            int32
	NtfyTopic          string
	UpdatedAt           pgtype.Timestamptz
}

type SavePushSubscriptionParams struct {
	UserID    string
	Endpoint  string
	P256dh    string
	Auth      string
	UserAgent string
}

type DeletePushSubscriptionParams struct {
	Endpoint string
	UserID   string
}

type UpsertNotificationPrefsParams struct {
	UserID              string
	WaterReminders      bool
	WaterIntervalHours  int32
	MealReminders       bool
	IfWindowReminders   bool
	StreakReminders     bool
	QuietStart          int32
	QuietEnd            int32
	NtfyTopic          string
}

const savePushSubscription = `INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, user_agent)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (endpoint) DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth, user_agent = EXCLUDED.user_agent`

func (q *Queries) SavePushSubscription(ctx context.Context, p SavePushSubscriptionParams) error {
	_, err := q.db.Exec(ctx, savePushSubscription, p.UserID, p.Endpoint, p.P256dh, p.Auth, p.UserAgent)
	return err
}

const deletePushSubscription = `DELETE FROM push_subscriptions WHERE endpoint = $1 AND user_id = $2`

func (q *Queries) DeletePushSubscription(ctx context.Context, p DeletePushSubscriptionParams) error {
	_, err := q.db.Exec(ctx, deletePushSubscription, p.Endpoint, p.UserID)
	return err
}

const getPushSubscriptionsByUser = `SELECT id, user_id, endpoint, p256dh, auth, user_agent, created_at FROM push_subscriptions WHERE user_id = $1`

func (q *Queries) GetPushSubscriptionsByUser(ctx context.Context, userID string) ([]PushSubscription, error) {
	rows, err := q.db.Query(ctx, getPushSubscriptionsByUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.UserAgent, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

const deletePushSubscriptionByEndpoint = `DELETE FROM push_subscriptions WHERE endpoint = $1`

func (q *Queries) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	_, err := q.db.Exec(ctx, deletePushSubscriptionByEndpoint, endpoint)
	return err
}

const getAllUsersWithPushSubscriptions = `SELECT DISTINCT user_id FROM push_subscriptions`

func (q *Queries) GetAllUsersWithPushSubscriptions(ctx context.Context) ([]string, error) {
	rows, err := q.db.Query(ctx, getAllUsersWithPushSubscriptions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

const getNotificationPrefs = `SELECT user_id, water_reminders, water_interval_hours, meal_reminders, if_window_reminders, streak_reminders, quiet_start, quiet_end, ntfy_topic, updated_at FROM notification_preferences WHERE user_id = $1`

func (q *Queries) GetNotificationPrefs(ctx context.Context, userID string) (NotificationPreferences, error) {
	row := q.db.QueryRow(ctx, getNotificationPrefs, userID)
	var p NotificationPreferences
	err := row.Scan(&p.UserID, &p.WaterReminders, &p.WaterIntervalHours, &p.MealReminders,
		&p.IfWindowReminders, &p.StreakReminders, &p.QuietStart, &p.QuietEnd, &p.NtfyTopic, &p.UpdatedAt)
	return p, err
}

const upsertNotificationPrefs = `INSERT INTO notification_preferences (user_id, water_reminders, water_interval_hours, meal_reminders, if_window_reminders, streak_reminders, quiet_start, quiet_end, ntfy_topic)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO UPDATE SET
    water_reminders = EXCLUDED.water_reminders, water_interval_hours = EXCLUDED.water_interval_hours,
    meal_reminders = EXCLUDED.meal_reminders, if_window_reminders = EXCLUDED.if_window_reminders,
    streak_reminders = EXCLUDED.streak_reminders, quiet_start = EXCLUDED.quiet_start,
    quiet_end = EXCLUDED.quiet_end, ntfy_topic = EXCLUDED.ntfy_topic, updated_at = NOW()
RETURNING user_id, water_reminders, water_interval_hours, meal_reminders, if_window_reminders, streak_reminders, quiet_start, quiet_end, ntfy_topic, updated_at`

func (q *Queries) UpsertNotificationPrefs(ctx context.Context, p UpsertNotificationPrefsParams) (NotificationPreferences, error) {
	row := q.db.QueryRow(ctx, upsertNotificationPrefs,
		p.UserID, p.WaterReminders, p.WaterIntervalHours, p.MealReminders,
		p.IfWindowReminders, p.StreakReminders, p.QuietStart, p.QuietEnd, p.NtfyTopic)
	var np NotificationPreferences
	err := row.Scan(&np.UserID, &np.WaterReminders, &np.WaterIntervalHours, &np.MealReminders,
		&np.IfWindowReminders, &np.StreakReminders, &np.QuietStart, &np.QuietEnd, &np.NtfyTopic, &np.UpdatedAt)
	return np, err
}

// GetLastWaterLog returns the time of the most recent water log for a user, or zero time if none today.
func (q *Queries) GetLastWaterLogTime(ctx context.Context, userID string) (time.Time, error) {
	row := q.db.QueryRow(ctx,
		`SELECT date FROM water_logs WHERE user_id = $1 ORDER BY date DESC LIMIT 1`, userID)
	var t time.Time
	err := row.Scan(&t)
	return t, err
}

// GetTodayMealCount returns how many meals the user has logged today.
func (q *Queries) GetTodayMealCount(ctx context.Context, userID string) (int, error) {
	row := q.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM meals WHERE user_id = $1 AND DATE(timestamp AT TIME ZONE 'UTC') = CURRENT_DATE`, userID)
	var count int
	err := row.Scan(&count)
	return count, err
}
