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

func GenerateIdentityQuotes(ctx context.Context, pool *pgxpool.Pool, aiClient ai.Client, _ *notify.Service) {
	slog.Info("scheduler: GenerateIdentityQuotes started")

	today := time.Now().UTC().Truncate(24 * time.Hour)
	todayStr := today.Format("2006-01-02")

	rows, err := pool.Query(ctx,
		`SELECT u.id FROM users u
		 WHERE u.verified = true
		   AND NOT EXISTS (
		     SELECT 1 FROM identity_quotes iq
		     WHERE iq.user_id = u.id AND iq.date = $1
		   )
		   AND EXISTS (
		     SELECT 1 FROM meals m WHERE m.user_id = u.id AND m.timestamp >= NOW() - INTERVAL '7 days'
		   )`,
		todayStr)
	if err != nil {
		slog.Error("scheduler: GenerateIdentityQuotes query users failed", "error", err)
		return
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err == nil {
			userIDs = append(userIDs, uid)
		}
	}

	updated := 0
	for _, uid := range userIDs {
		var aspiration string
		_ = pool.QueryRow(ctx,
			`SELECT COALESCE(identity_aspiration, '') FROM user_profiles WHERE user_id = $1`,
			uid).Scan(&aspiration)

		var streakDays int
		_ = pool.QueryRow(ctx,
			`SELECT COALESCE(streak_days, 0) FROM user_stats WHERE user_id = $1`,
			uid).Scan(&streakDays)

		var recentFoods string
		foodRows, err := pool.Query(ctx,
			`SELECT DISTINCT m.name FROM meals m WHERE m.user_id = $1 AND m.timestamp >= NOW() - INTERVAL '3 days' ORDER BY m.timestamp DESC LIMIT 5`,
			uid)
		if err == nil {
			var foods []string
			for foodRows.Next() {
				var f string
				if foodRows.Scan(&f) == nil {
					foods = append(foods, f)
				}
			}
			foodRows.Close()
			if len(foods) > 0 {
				recentFoods = "Recent foods: " + foods[0]
				for i := 1; i < len(foods); i++ {
					recentFoods += ", " + foods[i]
				}
			}
		}

		contextType := "daily"
		var mealsYesterday int
		_ = pool.QueryRow(ctx,
			`SELECT COUNT(*)::int FROM meals WHERE user_id = $1 AND timestamp::date = CURRENT_DATE - 1`,
			uid).Scan(&mealsYesterday)
		if mealsYesterday == 0 {
			contextType = "post_lapse"
		}

		var milestoneStreaks = []int{7, 14, 21, 30, 60, 90, 100}
		for _, ms := range milestoneStreaks {
			if streakDays == ms {
				contextType = "post_milestone"
				break
			}
		}

		systemPrompt := `You are a motivational coach for a nutrition tracking app called Joules. Generate a single personalized identity-based quote (1-2 sentences) that connects the user's health identity to their daily food choices. Be specific, warm, and empowering. Do not use clichés. Return ONLY the quote text, nothing else.`

		userMsg := fmt.Sprintf(
			"User's identity aspiration: %q. Current streak: %d days. %s. Context: %s. Generate a fresh identity-based motivational quote for today.",
			aspiration, streakDays, recentFoods, contextType,
		)

		quote, err := aiClient.Chat(systemPrompt, []ai.ChatMessage{
			{Role: "user", Content: userMsg},
		})
		if err != nil {
			slog.Error("scheduler: quote generation failed", "user_id", uid, "error", err)
			quote = "Every healthy choice you make today is a vote for the person you want to become."
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO identity_quotes (user_id, quote, date, context_type)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (user_id, date) DO UPDATE SET quote = $2, context_type = $4`,
			uid, quote, todayStr, contextType)
		if err != nil {
			slog.Error("scheduler: store quote failed", "user_id", uid, "error", err)
			continue
		}
		updated++
	}

	slog.Info("scheduler: GenerateIdentityQuotes complete", "users", len(userIDs), "generated", updated)
}
