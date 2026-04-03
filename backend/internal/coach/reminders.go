package coach

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReminderEntry struct {
	ID           string
	UserID       string
	Type         string
	Message      string
	ReminderTime string
	Enabled      bool
	CreatedAt    time.Time
}

func CreateReminder(ctx context.Context, pool *pgxpool.Pool, userID, reminderType, message, reminderTime string) (*ReminderEntry, error) {
	parsedTime, err := time.Parse("15:04", reminderTime)
	if err != nil {
		return nil, err
	}

	var r ReminderEntry
	err = pool.QueryRow(ctx,
		`INSERT INTO coach_reminders (user_id, type, message, reminder_time)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, type, message, reminder_time, enabled, created_at`,
		userID, reminderType, message, parsedTime.Format("15:04:05"),
	).Scan(&r.ID, &r.UserID, &r.Type, &r.Message, &r.ReminderTime, &r.Enabled, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func GetReminders(ctx context.Context, pool *pgxpool.Pool, userID string) ([]ReminderEntry, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, user_id, type, message, reminder_time, enabled, created_at
		 FROM coach_reminders
		 WHERE user_id = $1
		 ORDER BY reminder_time`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ReminderEntry
	for rows.Next() {
		var r ReminderEntry
		if err := rows.Scan(&r.ID, &r.UserID, &r.Type, &r.Message, &r.ReminderTime, &r.Enabled, &r.CreatedAt); err != nil {
			continue
		}
		entries = append(entries, r)
	}
	return entries, nil
}

func DeleteReminder(ctx context.Context, pool *pgxpool.Pool, userID, reminderID string) error {
	_, err := pool.Exec(ctx,
		"DELETE FROM coach_reminders WHERE id = $1 AND user_id = $2",
		reminderID, userID,
	)
	return err
}

func ToggleReminder(ctx context.Context, pool *pgxpool.Pool, userID, reminderID string, enabled bool) error {
	_, err := pool.Exec(ctx,
		"UPDATE coach_reminders SET enabled = $1 WHERE id = $2 AND user_id = $3",
		enabled, reminderID, userID,
	)
	return err
}
