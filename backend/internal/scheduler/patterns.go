package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/ai"
	"joules/internal/notify"
)

type behavioralPatterns struct {
	LateNightSnacking float64 `json:"late_night_snacking_pct"`
	WeekendAdherence  float64 `json:"weekend_adherence_pct"`
	WeekdayAdherence  float64 `json:"weekday_adherence_pct"`
	StressEatingDays  int     `json:"stress_eating_days"`
	PlateauDetected   bool    `json:"plateau_detected"`
	PlateauWeeks      int     `json:"plateau_weeks,omitempty"`
	ComputedAt        string  `json:"computed_at"`
}

func ComputeBehavioralPatterns(ctx context.Context, pool *pgxpool.Pool, _ ai.Client, _ *notify.Service) {
	slog.Info("scheduler: ComputeBehavioralPatterns started")

	rows, err := pool.Query(ctx,
		`SELECT us.user_id FROM user_stats us
		 WHERE EXISTS (SELECT 1 FROM meals m WHERE m.user_id = us.user_id AND m.timestamp >= NOW() - INTERVAL '30 days')`)
	if err != nil {
		slog.Error("scheduler: ComputeBehavioralPatterns query users failed", "error", err)
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

	now := time.Now().UTC()
	updated := 0

	for _, uid := range userIDs {
		patterns := behavioralPatterns{ComputedAt: now.Format(time.RFC3339)}
		periodStart := now.AddDate(0, 0, -30)

		// Late-night snacking: meals logged after 10pm as % of total meals
		var totalMeals, lateMeals int
		_ = pool.QueryRow(ctx,
			`SELECT COUNT(*)::int FROM meals WHERE user_id = $1 AND timestamp >= $2`,
			uid, periodStart).Scan(&totalMeals)
		if totalMeals > 0 {
			_ = pool.QueryRow(ctx,
				`SELECT COUNT(*)::int FROM meals WHERE user_id = $1 AND timestamp >= $2
				 AND EXTRACT(HOUR FROM timestamp) >= 22`,
				uid, periodStart).Scan(&lateMeals)
			patterns.LateNightSnacking = float64(lateMeals) / float64(totalMeals) * 100
		}

		// Weekend vs weekday adherence (days with meals / total days)
		var weekdayDays, weekendDays int
		var weekdayLogged, weekendLogged int
		_ = pool.QueryRow(ctx,
			`SELECT COUNT(DISTINCT d)::int FROM generate_series($2::date, $3::date, '1 day'::interval) d
			 WHERE EXTRACT(ISODOW FROM d) IN (1,2,3,4,5)`,
			uid, periodStart.Format("2006-01-02"), now.Format("2006-01-02")).Scan(&weekdayDays)
		_ = pool.QueryRow(ctx,
			`SELECT COUNT(DISTINCT d)::int FROM generate_series($2::date, $3::date, '1 day'::interval) d
			 WHERE EXTRACT(ISODOW FROM d) IN (6,7)`,
			uid, periodStart.Format("2006-01-02"), now.Format("2006-01-02")).Scan(&weekendDays)

		_ = pool.QueryRow(ctx,
			`SELECT COUNT(DISTINCT timestamp::date)::int FROM meals WHERE user_id = $1 AND timestamp >= $2
			 AND EXTRACT(ISODOW FROM timestamp) IN (1,2,3,4,5)`,
			uid, periodStart).Scan(&weekdayLogged)
		_ = pool.QueryRow(ctx,
			`SELECT COUNT(DISTINCT timestamp::date)::int FROM meals WHERE user_id = $1 AND timestamp >= $2
			 AND EXTRACT(ISODOW FROM timestamp) IN (6,7)`,
			uid, periodStart).Scan(&weekendLogged)

		if weekdayDays > 0 {
			patterns.WeekdayAdherence = float64(weekdayLogged) / float64(weekdayDays) * 100
		}
		if weekendDays > 0 {
			patterns.WeekendAdherence = float64(weekendLogged) / float64(weekendDays) * 100
		}

		// Stress eating: days with calorie variance > 50% from mean
		var daysWithHighVariance int
		_ = pool.QueryRow(ctx,
			`SELECT COUNT(*)::int FROM (
				SELECT timestamp::date AS d,
				       ABS(SUM(fi.calories) - (SELECT AVG(day_total) FROM (
				         SELECT SUM(fi2.calories) AS day_total FROM meals m2
				         JOIN food_items fi2 ON fi2.meal_id = m2.id
				         WHERE m2.user_id = $1 AND m2.timestamp >= $2
				         GROUP BY m2.timestamp::date
				       ) sub)) * 1.0 / NULLIF((SELECT AVG(day_total) FROM (
				         SELECT SUM(fi3.calories) AS day_total FROM meals m3
				         JOIN food_items fi3 ON fi3.meal_id = m3.id
				         WHERE m3.user_id = $1 AND m3.timestamp >= $2
				         GROUP BY m3.timestamp::date
				       ) sub2), 0) AS variance_ratio
				FROM meals m
				JOIN food_items fi ON fi.meal_id = m.id
				WHERE m.user_id = $1 AND m.timestamp >= $2
				GROUP BY m.timestamp::date
				HAVING variance_ratio > 0.5
			) sub3`,
			uid, periodStart).Scan(&daysWithHighVariance)
		patterns.StressEatingDays = daysWithHighVariance

		// Plateau detection: weight flat for 2+ weeks despite consistent logging
		var weightChange float64
		var weeks int
		_ = pool.QueryRow(ctx,
			`SELECT ABS(w2.weight - w1.weight),
			        (w2.date - w1.date)::int / 7
		 FROM weight_logs w1, weight_logs w2
		 WHERE w1.user_id = $1 AND w2.user_id = $1
		   AND w1.date = (SELECT MIN(date) FROM weight_logs WHERE user_id = $1 AND date >= NOW() - INTERVAL '30 days')
		   AND w2.date = (SELECT MAX(date) FROM weight_logs WHERE user_id = $1 AND date >= NOW() - INTERVAL '30 days')`,
			uid).Scan(&weightChange, &weeks)

		var mealsLast14Days int
		_ = pool.QueryRow(ctx,
			`SELECT COUNT(DISTINCT timestamp::date)::int FROM meals WHERE user_id = $1 AND timestamp >= NOW() - INTERVAL '14 days'`,
			uid).Scan(&mealsLast14Days)

		if weeks >= 2 && weightChange < 0.5 && mealsLast14Days >= 10 {
			patterns.PlateauDetected = true
			patterns.PlateauWeeks = weeks
		}

		jsonData, err := json.Marshal(patterns)
		if err != nil {
			slog.Error("scheduler: marshal patterns failed", "user_id", uid, "error", err)
			continue
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO behavioral_insights (user_id, patterns, computed_at)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (user_id) DO UPDATE SET patterns = $2, computed_at = $3`,
			uid, jsonData, now)
		if err != nil {
			slog.Error("scheduler: upsert behavioral insights failed", "user_id", uid, "error", err)
			continue
		}
		updated++
	}

	slog.Info("scheduler: ComputeBehavioralPatterns complete", "users", len(userIDs), "updated", updated)
}

// Ensure behavioral_insights table name is referenced correctly for schema
var _ = fmt.Sprintf("behavioral_insights")
