package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/ai"
	"joules/internal/notify"
)

func DetectFreshStarts(ctx context.Context, pool *pgxpool.Pool, _ ai.Client, notifySvc *notify.Service) {
	slog.Info("scheduler: DetectFreshStarts started")

	now := time.Now().UTC()
	weekday := now.Weekday()
	dayOfMonth := now.Day()

	if weekday == time.Monday {
		sendFreshStartNotification(ctx, pool, notifySvc,
			"fresh-start-monday",
			"New Week, Fresh Momentum",
			"A brand new week — a perfect time to build on your progress. What's one healthy choice you'll make today?",
		)
	}

	if dayOfMonth == 1 {
		sendFreshStartNotification(ctx, pool, notifySvc,
			"fresh-start-monthly",
			"Monthly Progress Review",
			fmt.Sprintf("It's the 1st of %s! Take a moment to review your progress and set your intentions for the month ahead.", now.Format("January")),
		)
	}

	detectInactiveUsers(ctx, pool, notifySvc)
}

func sendFreshStartNotification(ctx context.Context, pool *pgxpool.Pool, notifySvc *notify.Service, tag, title, body string) {
	rows, err := pool.Query(ctx,
		`SELECT u.id FROM users u
		 WHERE u.verified = true
		   AND EXISTS (SELECT 1 FROM meals m WHERE m.user_id = u.id AND m.timestamp >= NOW() - INTERVAL '30 days')`)
	if err != nil {
		slog.Error("scheduler: fresh start query failed", "error", err)
		return
	}
	defer rows.Close()

	sent := 0
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			continue
		}

		var alreadySent bool
		_ = pool.QueryRow(ctx,
			`SELECT EXISTS (
			 SELECT 1 FROM notification_log
			 WHERE user_id = $1 AND tag = $2 AND created_at::date = CURRENT_DATE
			)`,
			uid, tag).Scan(&alreadySent)
		if alreadySent {
			continue
		}

		if notifySvc != nil {
			notifySvc.SendToUser(ctx, uid, notify.Payload{
				Title: title,
				Body:  body,
				URL:   "/dashboard",
				Tag:   tag,
			})
		}
		sent++
	}

	slog.Info("scheduler: fresh start notifications sent", "tag", tag, "count", sent)
}

func detectInactiveUsers(ctx context.Context, pool *pgxpool.Pool, notifySvc *notify.Service) {
	rows, err := pool.Query(ctx,
		`SELECT u.id FROM users u
		 WHERE u.verified = true
		   AND EXISTS (SELECT 1 FROM meals m WHERE m.user_id = u.id)
		   AND NOT EXISTS (SELECT 1 FROM meals m WHERE m.user_id = u.id AND m.timestamp >= NOW() - INTERVAL '7 days')
		   AND NOT EXISTS (
		     SELECT 1 FROM notification_log nl
		     WHERE nl.user_id = u.id AND nl.tag = 'fresh-start-reengage'
		       AND nl.created_at >= NOW() - INTERVAL '7 days'
		   )`)
	if err != nil {
		slog.Error("scheduler: inactive users query failed", "error", err)
		return
	}
	defer rows.Close()

	sent := 0
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			continue
		}

		if notifySvc != nil {
			notifySvc.SendToUser(ctx, uid, notify.Payload{
				Title: "We Miss You",
				Body:  "Your health journey matters. Even a small step today — logging one meal — gets you back on track. We're here for you.",
				URL:   "/log",
				Tag:   "fresh-start-reengage",
			})
		}
		sent++
	}

	slog.Info("scheduler: re-engagement notifications sent", "count", sent)
}
