package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/ai"
	"joules/internal/notify"
)

func UpdateHabitPhases(ctx context.Context, pool *pgxpool.Pool, _ ai.Client, _ *notify.Service) {
	slog.Info("scheduler: UpdateHabitPhases started")

	rows, err := pool.Query(ctx,
		`SELECT us.user_id, us.current_phase, us.phase_updated_at,
		        (SELECT MIN(m.timestamp) FROM meals m WHERE m.user_id = us.user_id) AS first_meal
		 FROM user_stats us`)
	if err != nil {
		slog.Error("scheduler: UpdateHabitPhases query failed", "error", err)
		return
	}
	defer rows.Close()

	now := time.Now().UTC()
	updated := 0

	for rows.Next() {
		var userID string
		var currentPhase string
		var phaseUpdatedAt *time.Time
		var firstMeal *time.Time

		if err := rows.Scan(&userID, &currentPhase, &phaseUpdatedAt, &firstMeal); err != nil {
			slog.Error("scheduler: UpdateHabitPhases scan failed", "error", err)
			continue
		}

		if firstMeal == nil {
			continue
		}

		totalDays := int(now.Sub(*firstMeal).Hours()/24) + 1

		var newPhase string
		switch {
		case totalDays <= 30:
			newPhase = "scaffolding"
		case totalDays <= 66:
			newPhase = "identity_building"
		case totalDays <= 90:
			newPhase = "intrinsic"
		default:
			newPhase = "maintenance"
		}

		if newPhase == currentPhase {
			continue
		}

		_, err := pool.Exec(ctx,
			`UPDATE user_stats SET current_phase = $1, phase_updated_at = $2 WHERE user_id = $3`,
			newPhase, now, userID)
		if err != nil {
			slog.Error("scheduler: UpdateHabitPhases update failed", "user_id", userID, "error", err)
			continue
		}

		slog.Info("scheduler: phase updated",
			"user_id", userID,
			"old_phase", currentPhase,
			"new_phase", newPhase,
			"total_days", totalDays)
		updated++
	}

	slog.Info("scheduler: UpdateHabitPhases complete", "updated", updated)
}
