package habits

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/auth"
	"joules/internal/db/sqlc"
)

type Handler struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewHandler(q *sqlc.Queries, pool *pgxpool.Pool) *Handler {
	return &Handler{q: q, pool: pool}
}

type apiResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	slog.Error("habits error", "status", status, "error", err)
	msg := err.Error()
	if status >= 500 {
		msg = "internal server error"
	}
	writeJSON(w, status, apiResponse{Error: msg})
}

func getUserID(r *http.Request) (string, error) {
	userID, ok := r.Context().Value(auth.ContextUserID).(string)
	if !ok {
		return "", fmt.Errorf("unauthorized")
	}
	return userID, nil
}

// HabitSummary is the response for GET /habits/summary.
type HabitSummary struct {
	TotalPoints      int32   `json:"total_points"`
	Level            int     `json:"level"`
	LevelName        string  `json:"level_name"`
	LevelProgressPct float64 `json:"level_progress_pct"`
	NextLevelAt      int     `json:"next_level_at"`
	StreakDays       int32   `json:"streak_days"`
	PetMood          string  `json:"pet_mood"`
	TodayPoints      int     `json:"today_points"`
	TodayCheckedIn   bool    `json:"today_checked_in"`
}

type levelDef struct {
	name   string
	minPts int
	maxPts int // exclusive; -1 means no cap (Legend)
}

var levels = []levelDef{
	{name: "Beginner", minPts: 0, maxPts: 100},
	{name: "Getting Started", minPts: 100, maxPts: 250},
	{name: "Consistent", minPts: 250, maxPts: 500},
	{name: "Dedicated", minPts: 500, maxPts: 1000},
	{name: "Champion", minPts: 1000, maxPts: 2000},
	{name: "Legend", minPts: 2000, maxPts: -1},
}

func computeLevel(totalPoints int32) (level int, name string, progressPct float64, nextAt int) {
	pts := int(totalPoints)
	for i, l := range levels {
		if l.maxPts == -1 || pts < l.maxPts {
			level = i + 1
			name = l.name
			if l.maxPts == -1 {
				progressPct = 100
				nextAt = -1
			} else {
				span := l.maxPts - l.minPts
				progressPct = float64(pts-l.minPts) / float64(span) * 100
				nextAt = l.maxPts
			}
			return
		}
	}
	// Fallback: max level
	last := levels[len(levels)-1]
	return len(levels), last.name, 100, -1
}

// petMood returns a mood string based on streak and last active date.
func petMood(streakDays int32, lastActive *time.Time) string {
	if lastActive == nil {
		return "sleeping"
	}
	daysSince := int(time.Since(*lastActive).Hours() / 24)
	if daysSince >= 7 {
		return "sleeping"
	}
	if daysSince >= 4 || streakDays == 0 {
		return "sad"
	}
	if streakDays >= 7 {
		return "thriving"
	}
	if streakDays >= 3 {
		return "happy"
	}
	return "okay"
}

// todayPoints tallies how many points the user earned today based on activity.
// Scoring: meal logged +5, >=3 meals +10 bonus, exercise +10,
// water >=2000ml +10, steps >=5k +10, steps >=10k +10 more.
func (h *Handler) todayPoints(ctx context.Context, userID string, today time.Time) int {
	todayStr := today.Format("2006-01-02")
	pts := 0

	// Meals logged today
	var mealCount int
	row := h.pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM meals WHERE user_id = $1 AND timestamp::date = $2`,
		userID, todayStr)
	if err := row.Scan(&mealCount); err == nil && mealCount > 0 {
		pts += 5
		if mealCount >= 3 {
			pts += 10
		}
	}

	// Exercise logged today
	var exerciseCount int
	row = h.pool.QueryRow(ctx,
		`SELECT COUNT(*)::int FROM exercises WHERE user_id = $1 AND timestamp::date = $2`,
		userID, todayStr)
	if err := row.Scan(&exerciseCount); err == nil && exerciseCount > 0 {
		pts += 10
	}

	// Water intake today
	var waterMl int
	row = h.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_ml), 0)::int FROM water_logs WHERE user_id = $1 AND date::date = $2`,
		userID, todayStr)
	if err := row.Scan(&waterMl); err == nil && waterMl >= 2000 {
		pts += 10
	}

	// Steps today
	var steps int
	row = h.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(step_count), 0)::int FROM step_logs WHERE user_id = $1 AND date = $2`,
		userID, todayStr)
	if err := row.Scan(&steps); err == nil {
		if steps >= 5000 {
			pts += 10
		}
		if steps >= 10000 {
			pts += 10
		}
	}

	return pts
}

// GetSummary handles GET /habits/summary.
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	ctx := r.Context()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	stats, err := h.q.GetUserStats(ctx, userID)
	if err != nil && !isNoRows(err) {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get user stats: %w", err))
		return
	}

	// Determine last active time pointer for petMood
	var lastActivePtr *time.Time
	if stats.LastActiveDate.Valid {
		t := stats.LastActiveDate.Time
		lastActivePtr = &t
	}

	todayCheckedIn := stats.LastActiveDate.Valid &&
		stats.LastActiveDate.Time.Equal(today)

	level, levelName, progressPct, nextAt := computeLevel(stats.TotalPoints)
	todayPts := h.todayPoints(ctx, userID, today)

	summary := HabitSummary{
		TotalPoints:      stats.TotalPoints,
		Level:            level,
		LevelName:        levelName,
		LevelProgressPct: progressPct,
		NextLevelAt:      nextAt,
		StreakDays:       stats.StreakDays,
		PetMood:          petMood(stats.StreakDays, lastActivePtr),
		TodayPoints:      todayPts,
		TodayCheckedIn:   todayCheckedIn,
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: summary})
}

// Checkin handles POST /habits/checkin.
// Idempotent: if already checked in today, returns current stats unchanged.
func (h *Handler) Checkin(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	ctx := r.Context()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	stats, err := h.q.GetUserStats(ctx, userID)
	if err != nil && !isNoRows(err) {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get user stats: %w", err))
		return
	}

	// Idempotency: already checked in today
	if stats.LastActiveDate.Valid && stats.LastActiveDate.Time.Equal(today) {
		var lastActivePtr *time.Time
		if stats.LastActiveDate.Valid {
			t := stats.LastActiveDate.Time
			lastActivePtr = &t
		}
		level, levelName, progressPct, nextAt := computeLevel(stats.TotalPoints)
		todayPts := h.todayPoints(ctx, userID, today)
		writeJSON(w, http.StatusOK, apiResponse{Data: HabitSummary{
			TotalPoints:      stats.TotalPoints,
			Level:            level,
			LevelName:        levelName,
			LevelProgressPct: progressPct,
			NextLevelAt:      nextAt,
			StreakDays:       stats.StreakDays,
			PetMood:          petMood(stats.StreakDays, lastActivePtr),
			TodayPoints:      todayPts,
			TodayCheckedIn:   true,
		}})
		return
	}

	// Compute today's earned points
	todayPts := h.todayPoints(ctx, userID, today)

	// Streak logic
	var newStreak int32
	if stats.LastActiveDate.Valid && stats.LastActiveDate.Time.Equal(yesterday) {
		// Consecutive day — extend streak
		newStreak = stats.StreakDays + 1
	} else {
		// Gap or first ever checkin — reset to 1
		newStreak = 1
	}

	// Streak bonus: +1 pt per streak day, capped at 30
	streakBonus := int(newStreak)
	if streakBonus > 30 {
		streakBonus = 30
	}

	newPoints := stats.TotalPoints + int32(todayPts) + int32(streakBonus)

	updated, err := h.q.UpsertUserStats(ctx, sqlc.UpsertUserStatsParams{
		UserID:         userID,
		TotalPoints:    newPoints,
		StreakDays:     newStreak,
		LastActiveDate: pgtype.Date{Time: today, Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("upsert user stats: %w", err))
		return
	}

	t := updated.LastActiveDate.Time
	level, levelName, progressPct, nextAt := computeLevel(updated.TotalPoints)
	writeJSON(w, http.StatusOK, apiResponse{Data: HabitSummary{
		TotalPoints:      updated.TotalPoints,
		Level:            level,
		LevelName:        levelName,
		LevelProgressPct: progressPct,
		NextLevelAt:      nextAt,
		StreakDays:       updated.StreakDays,
		PetMood:          petMood(updated.StreakDays, &t),
		TodayPoints:      todayPts + streakBonus,
		TodayCheckedIn:   true,
	}})
}

type phaseResponse struct {
	Phase                 string  `json:"phase"`
	DaysInPhase           int     `json:"days_in_phase"`
	TotalDays             int     `json:"total_days"`
	NextPhaseDate         string  `json:"next_phase_date"`
	ConsistencyPct        float64 `json:"consistency_percentage"`
	GraceDaysUsedThisWeek int     `json:"grace_days_used_this_week"`
	GraceDaysMaxPerWeek   int     `json:"grace_days_max_per_week"`
}

var phaseOrder = []string{"scaffolding", "building", "strengthening", "thriving"}
var phaseDuration = map[string]int{
	"scaffolding":   14,
	"building":      14,
	"strengthening": 14,
	"thriving":      -1,
}

func (h *Handler) GetPhase(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	ctx := r.Context()
	now := time.Now().UTC()

	var currentPhase string
	var phaseUpdatedAt *time.Time
	err = h.pool.QueryRow(ctx,
		`SELECT current_phase, phase_updated_at FROM user_stats WHERE user_id = $1`,
		userID).Scan(&currentPhase, &phaseUpdatedAt)
	if err != nil && !isNoRows(err) {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get phase: %w", err))
		return
	}
	if currentPhase == "" {
		currentPhase = "scaffolding"
	}

	var daysInPhase int
	if phaseUpdatedAt != nil {
		daysInPhase = int(now.Sub(*phaseUpdatedAt).Hours() / 24)
	}

	var firstMealDate *time.Time
	_ = h.pool.QueryRow(ctx,
		`SELECT MIN(timestamp) FROM meals WHERE user_id = $1`,
		userID).Scan(&firstMealDate)

	totalDays := 1
	if firstMealDate != nil {
		totalDays = int(now.Sub(*firstMealDate).Hours()/24) + 1
	}

	var daysWithMeals int
	_ = h.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT timestamp::date)::int FROM meals WHERE user_id = $1 AND timestamp >= $2`,
		userID, firstMealDate).Scan(&daysWithMeals)

	consistencyPct := 0.0
	if totalDays > 0 {
		consistencyPct = float64(daysWithMeals) / float64(totalDays) * 100
	}

	dur := phaseDuration[currentPhase]
	nextPhaseDate := ""
	if dur > 0 && phaseUpdatedAt != nil {
		nextPhaseDate = phaseUpdatedAt.AddDate(0, 0, dur).Format(time.RFC3339)
	}

	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weekStartStr := weekStart.Format("2006-01-02")

	var graceUsed int
	var graceMax int
	graceMax = 2
	_ = h.pool.QueryRow(ctx,
		`SELECT COALESCE(days_used, 0), COALESCE(max_per_week, 2) FROM grace_days WHERE user_id = $1 AND week_start = $2`,
		userID, weekStartStr).Scan(&graceUsed, &graceMax)

	writeJSON(w, http.StatusOK, apiResponse{Data: phaseResponse{
		Phase:                 currentPhase,
		DaysInPhase:           daysInPhase,
		TotalDays:             totalDays,
		NextPhaseDate:         nextPhaseDate,
		ConsistencyPct:        consistencyPct,
		GraceDaysUsedThisWeek: graceUsed,
		GraceDaysMaxPerWeek:   graceMax,
	}})
}

func isNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
