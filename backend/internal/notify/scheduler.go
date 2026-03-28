package notify

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"joule/internal/db/sqlc"
)

// StartScheduler runs the notification scheduler in a background goroutine.
// It ticks every 15 minutes and evaluates what notifications to send each user.
func (s *Service) StartScheduler(ctx context.Context) {
	slog.Info("notify: scheduler started")
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	// Run once immediately on startup (after a short delay so DB is warm)
	go func() {
		time.Sleep(30 * time.Second)
		s.runSchedulerTick(ctx)
	}()

	for {
		select {
		case <-ticker.C:
			s.runSchedulerTick(ctx)
		case <-ctx.Done():
			slog.Info("notify: scheduler stopped")
			return
		}
	}
}

func (s *Service) runSchedulerTick(ctx context.Context) {
	userIDs, err := s.q.GetAllUsersWithPushSubscriptions(ctx)
	if err != nil {
		slog.Error("notify: scheduler get users failed", "error", err)
		return
	}

	now := time.Now().UTC()
	slog.Debug("notify: scheduler tick", "users", len(userIDs), "time", now.Format("15:04"))

	for _, uid := range userIDs {
		prefs, err := s.q.GetNotificationPrefs(ctx, uid)
		if err != nil {
			// User has no prefs saved yet — use defaults, all enabled
			prefs = sqlc.NotificationPreferences{
				WaterReminders:     true,
				WaterIntervalHours: 2,
				MealReminders:      true,
				IfWindowReminders:  true,
				StreakReminders:    true,
				QuietStart:         22,
				QuietEnd:           8,
			}
		}

		if isQuietHours(now, prefs) {
			continue
		}

		s.checkWaterReminder(ctx, uid, now, prefs)
		s.checkMealReminder(ctx, uid, now, prefs)
		s.checkIFWindowReminder(ctx, uid, now, prefs)
		s.checkStreakReminder(ctx, uid, now, prefs)
	}
}

// isQuietHours returns true if the current hour falls in the user's quiet window.
func isQuietHours(now time.Time, prefs sqlc.NotificationPreferences) bool {
	h := int32(now.Hour())
	qs, qe := prefs.QuietStart, prefs.QuietEnd
	if qs <= qe {
		// e.g. quiet 1–6: straightforward range
		return h >= qs && h < qe
	}
	// Wraps midnight, e.g. quiet 22–8: active from 22 to midnight and 0 to 8
	return h >= qs || h < qe
}

// checkWaterReminder sends a water reminder if the user hasn't logged water
// within their configured interval.
func (s *Service) checkWaterReminder(ctx context.Context, userID string, now time.Time, prefs sqlc.NotificationPreferences) {
	if !prefs.WaterReminders {
		return
	}

	lastLog, err := s.q.GetLastWaterLogTime(ctx, userID)
	if err != nil {
		// No logs at all — only remind once mid-morning
		if now.Hour() == 10 {
			s.SendToUser(ctx, userID, Payload{
				Title: "💧 Stay Hydrated",
				Body:  "You haven't logged any water yet today. Start with a glass!",
				URL:   "/dashboard",
				Tag:   "water",
			})
		}
		return
	}

	interval := time.Duration(prefs.WaterIntervalHours) * time.Hour
	if now.Sub(lastLog) >= interval {
		s.SendToUser(ctx, userID, Payload{
			Title: "💧 Water Reminder",
			Body:  fmt.Sprintf("It's been %dh since your last water log. Time for a glass!", prefs.WaterIntervalHours),
			URL:   "/dashboard",
			Tag:   "water",
		})
	}
}

// checkMealReminder nudges the user at noon if they haven't logged anything yet.
func (s *Service) checkMealReminder(ctx context.Context, userID string, now time.Time, prefs sqlc.NotificationPreferences) {
	if !prefs.MealReminders {
		return
	}
	// Only fire once, at noon (12:00–12:14 window)
	if now.Hour() != 12 {
		return
	}

	count, err := s.q.GetTodayMealCount(ctx, userID)
	if err != nil || count > 0 {
		return
	}

	s.SendToUser(ctx, userID, Payload{
		Title: "🍽️ Log Your Meals",
		Body:  "You haven't logged any food yet today. Keep your streak going!",
		URL:   "/log",
		Tag:   "meal-reminder",
	})
}

// checkIFWindowReminder sends alerts when an IF eating window is about to open or close.
func (s *Service) checkIFWindowReminder(ctx context.Context, userID string, now time.Time, prefs sqlc.NotificationPreferences) {
	if !prefs.IfWindowReminders {
		return
	}

	goals, err := s.q.GetGoals(ctx, userID)
	if err != nil {
		return
	}

	var windowHours int
	switch goals.DietPlan {
	case "intermittent_fasting":
		// Default to 16:8 if generic IF
		windowHours = 8
	default:
		return // Not an IF plan
	}

	// Check the fasting_window field if available
	if goals.FastingWindow != nil {
		switch *goals.FastingWindow {
		case "16:8":
			windowHours = 8
		case "18:6":
			windowHours = 6
		case "20:4":
			windowHours = 4
		case "omad":
			windowHours = 1
		}
	}

	fastHours := 24 - windowHours
	// Assume eating window starts at 12:00 by default
	// TODO: make this configurable per-user
	windowStartHour := 12
	windowEndHour := windowStartHour + windowHours
	if windowEndHour > 23 {
		windowEndHour -= 24
	}

	h := now.Hour()
	m := now.Minute()

	// 30 minutes before window opens
	openWarningHour := windowStartHour - 1
	if openWarningHour < 0 {
		openWarningHour += 24
	}
	if h == openWarningHour && m < 15 {
		s.SendToUser(ctx, userID, Payload{
			Title: "🍽️ Eating Window Opens Soon",
			Body:  fmt.Sprintf("Your %dh eating window opens in 30 minutes. Plan your first meal!", windowHours),
			URL:   "/log",
			Tag:   "if-window",
		})
	}

	// 30 minutes before window closes
	closeWarningHour := windowEndHour - 1
	if closeWarningHour < 0 {
		closeWarningHour += 24
	}
	if h == closeWarningHour && m >= 45 {
		s.SendToUser(ctx, userID, Payload{
			Title: "⏰ Eating Window Closes Soon",
			Body:  fmt.Sprintf("You have ~%d minutes left in your eating window. Last chance to log!", fastHours),
			URL:   "/log",
			Tag:   "if-window-close",
		})
	}
}

// checkStreakReminder fires at 21:00 if the user hasn't logged any meals today.
func (s *Service) checkStreakReminder(ctx context.Context, userID string, now time.Time, prefs sqlc.NotificationPreferences) {
	if !prefs.StreakReminders {
		return
	}
	if now.Hour() != 21 {
		return
	}

	count, err := s.q.GetTodayMealCount(ctx, userID)
	if err != nil || count > 0 {
		return
	}

	s.SendToUser(ctx, userID, Payload{
		Title: "🔥 Keep Your Streak Alive",
		Body:  "You haven't logged today yet. Log a meal before midnight to keep your streak!",
		URL:   "/log",
		Tag:   "streak",
	})
}
