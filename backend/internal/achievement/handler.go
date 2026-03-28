package achievement

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"joule/internal/auth"
	"joule/internal/db/sqlc"
)


type Handler struct {
	q *sqlc.Queries
}

func NewHandler(q *sqlc.Queries) *Handler {
	return &Handler{q: q}
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
	slog.Error("request error", "status", status, "error", err)
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

func getUserID(r *http.Request) string {
	return r.Context().Value(auth.ContextUserID).(string)
}

type achievementResponse struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	UnlockedAt  string `json:"unlocked_at"`
}

func (h *Handler) GetAchievements(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	achievements, err := h.q.GetAchievements(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := make([]achievementResponse, 0, len(achievements))
	for _, a := range achievements {
		resp = append(resp, achievementResponse{
			ID:          a.ID,
			Type:        a.Type,
			Title:       a.Title,
			Description: a.Description,
			UnlockedAt:  a.UnlockedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *Handler) CheckAchievements(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserID(r)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)

	var summary sqlc.GetDailySummaryRow
	var goals sqlc.UserGoal
	hasSummary := false
	hasGoals := false

	if s, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{UserID: userID, Timestamp: today}); err == nil {
		summary = s
		hasSummary = true
	}
	if g, err := h.q.GetGoals(ctx, userID); err == nil {
		goals = g
		hasGoals = true
	}

	meals, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: today})
	weights, _ := h.q.GetWeightHistory(ctx, sqlc.GetWeightHistoryParams{UserID: userID, Date: today.AddDate(0, 0, -365), Date_2: today})
	exercises, _ := h.q.GetExercisesByDate(ctx, sqlc.GetExercisesByDateParams{UserID: userID, Timestamp: today})
	water, _ := h.q.GetWaterByDate(ctx, sqlc.GetWaterByDateParams{UserID: userID, Date: today})
	chatHistory, _ := h.q.GetCoachHistory(ctx, userID)
	mealsYesterday, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: yesterday})
	mealsTwoDaysAgo, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: twoDaysAgo})

	type check struct {
		Type        string
		Title       string
		Description string
		Met         bool
	}

	checks := []check{
		{"first_meal", "First Bite", "Logged your first meal", len(meals) > 0},
		{"first_weight", "Scale It", "Logged your weight for the first time", len(weights) > 0},
		{"first_exercise", "Getting Active", "Logged your first exercise", len(exercises) > 0},
		{"first_water", "Hydration Start", "Started tracking water intake", water > 0},
		{"first_chat", "Coach Connection", "Had your first chat with the coach", len(chatHistory) > 0},
		{"streak_3", "3-Day Streak", "Logged meals for 3 consecutive days", len(meals) > 0 && len(mealsYesterday) > 0 && len(mealsTwoDaysAgo) > 0},
		{"calorie_goal", "On Target", "Hit your daily calorie goal", hasSummary && hasGoals && goals.DailyCalorieTarget > 0 && summary.TotalCalories >= goals.DailyCalorieTarget},
		{"water_goal", "Hydrated", "Drank 2500ml+ in a day", hasSummary && summary.TotalWaterMl >= 2500},
	}

	for _, c := range checks {
		if c.Met {
			h.q.UnlockAchievement(ctx, sqlc.UnlockAchievementParams{
				UserID:      userID,
				Type:        c.Type,
				Title:       c.Title,
				Description: c.Description,
			})
		}
	}

	achievements, err := h.q.GetAchievements(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := make([]achievementResponse, 0, len(achievements))
	for _, a := range achievements {
		resp = append(resp, achievementResponse{
			ID:          a.ID,
			Type:        a.Type,
			Title:       a.Title,
			Description: a.Description,
			UnlockedAt:  a.UnlockedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}
