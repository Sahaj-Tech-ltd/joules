package achievement

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"joules/internal/auth"
	"joules/internal/db/sqlc"
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

type achievementResponse struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Category        string `json:"category"`
	UnlockedAt      string `json:"unlocked_at"`
	ProgressCurrent int    `json:"progress_current"`
	ProgressTarget  int    `json:"progress_target"`
}

func (h *Handler) GetAchievements(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	achievements, err := h.q.GetAchievements(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	resp := make([]achievementResponse, 0, len(achievements))
	for _, a := range achievements {
		resp = append(resp, achievementResponse{
			ID:              a.ID,
			Type:            a.Type,
			Title:           a.Title,
			Description:     a.Description,
			Category:        a.Category,
			UnlockedAt:      a.UnlockedAt.Format(time.RFC3339),
			ProgressCurrent: int(a.ProgressCurrent),
			ProgressTarget:  int(a.ProgressTarget),
		})
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *Handler) CheckAchievements(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)

	var summary sqlc.GetDailySummaryRow
	var goals sqlc.UserGoal
	hasSummary := false
	hasGoals := false

	if s, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{UserID: userID, Timestamp: today, Column3: "UTC"}); err == nil {
		summary = s
		hasSummary = true
	}
	if g, err := h.q.GetGoals(ctx, userID); err == nil {
		goals = g
		hasGoals = true
	}

	threeDaysAgo := today.AddDate(0, 0, -3)
	fourDaysAgo := today.AddDate(0, 0, -4)
	fiveDaysAgo := today.AddDate(0, 0, -5)
	sixDaysAgo := today.AddDate(0, 0, -6)

	meals, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: today})
	weights, _ := h.q.GetWeightHistory(ctx, sqlc.GetWeightHistoryParams{UserID: userID, Date: today.AddDate(0, 0, -365), Date_2: today})
	exercises, _ := h.q.GetExercisesByDate(ctx, sqlc.GetExercisesByDateParams{UserID: userID, Timestamp: today, Column3: "UTC"})
	water, _ := h.q.GetWaterByDate(ctx, sqlc.GetWaterByDateParams{UserID: userID, Date: today})
	chatHistory, _ := h.q.GetCoachHistory(ctx, userID)
	mealsYesterday, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: yesterday})
	mealsTwoDaysAgo, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: twoDaysAgo})
	mealsThreeDaysAgo, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: threeDaysAgo})
	mealsFourDaysAgo, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: fourDaysAgo})
	mealsFiveDaysAgo, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: fiveDaysAgo})
	mealsSixDaysAgo, _ := h.q.GetMealsByDate(ctx, sqlc.GetMealsByDateParams{UserID: userID, Timestamp: sixDaysAgo})

	streak7 := len(meals) > 0 && len(mealsYesterday) > 0 && len(mealsTwoDaysAgo) > 0 &&
		len(mealsThreeDaysAgo) > 0 && len(mealsFourDaysAgo) > 0 && len(mealsFiveDaysAgo) > 0 && len(mealsSixDaysAgo) > 0

	type check struct {
		Type            string
		Title           string
		Description     string
		Category        string
		Met             bool
		ProgressCurrent int
		ProgressTarget  int
	}

	checks := []check{
		{"first_meal", "First Bite", "Logged your first meal", "meals", len(meals) > 0, 0, 0},
		{"first_weight", "Scale It", "Logged your weight for the first time", "weight", len(weights) > 0, 0, 0},
		{"first_exercise", "Getting Active", "Logged your first exercise", "exercise", len(exercises) > 0, 0, 0},
		{"first_water", "Hydration Start", "Started tracking water intake", "water", water > 0, 0, 0},
		{"first_chat", "Coach Connection", "Had your first chat with the coach", "coach", len(chatHistory) > 0, 0, 0},

		{"streak_3", "3-Day Streak", "Logged meals for 3 consecutive days", "consistency", len(meals) > 0 && len(mealsYesterday) > 0 && len(mealsTwoDaysAgo) > 0, 0, 0},
		{"streak_7", "Week Warrior", "Logged meals for 7 consecutive days", "consistency", streak7, 0, 0},
		{"streak_14", "Two-Week Titan", "Logged meals for 14 consecutive days", "consistency", false, 0, 14},
		{"streak_30", "Monthly Master", "Logged meals for 30 consecutive days", "consistency", false, 0, 30},
		{"streak_100", "Centurion", "Logged meals for 100 consecutive days", "consistency", false, 0, 0},

		{"calorie_goal", "On Target", "Hit your daily calorie goal", "nutrition", hasSummary && hasGoals && goals.DailyCalorieTarget > 0 && summary.TotalCalories >= goals.DailyCalorieTarget, 0, 0},
		{"protein_goal", "Protein Power", "Hit your daily protein goal", "nutrition", hasSummary && hasGoals && goals.DailyProteinG > 0 && summary.TotalProtein >= float64(goals.DailyProteinG), 0, 0},
		{"perfect_day", "Perfect Day", "Hit both calorie and protein goals in one day", "nutrition", hasSummary && hasGoals && goals.DailyCalorieTarget > 0 && summary.TotalCalories >= goals.DailyCalorieTarget && goals.DailyProteinG > 0 && summary.TotalProtein >= float64(goals.DailyProteinG), 0, 0},
		{"low_carb_day", "Low Carb Day", "Stayed under 50g carbs for the day", "nutrition", hasSummary && len(meals) > 0 && summary.TotalCarbs < 50, 0, 0},
		{"high_protein_day", "Protein Beast", "Consumed over 150g protein in a day", "nutrition", hasSummary && summary.TotalProtein >= 150, 0, 0},
		{"fiber_champion", "Fiber Champion", "Consumed over 30g fiber in a day", "nutrition", hasSummary && summary.TotalFiber >= 30, 0, 0},

		{"water_goal", "Hydrated", "Drank 2500ml+ in a day", "water", hasSummary && summary.TotalWaterMl >= 2500, 0, 0},
		{"water_3l", "Waterfall", "Drank 3000ml+ in a day", "water", hasSummary && summary.TotalWaterMl >= 3000, 0, 0},
		{"water_4l", "Hydration Hero", "Drank 4000ml+ in a day", "water", hasSummary && summary.TotalWaterMl >= 4000, 0, 0},

		{"exercise_1", "First Burn", "Burned calories through exercise", "exercise", len(exercises) > 0, 0, 0},
		{"exercise_5", "Fifth Gear", "Completed 5 exercise sessions", "exercise", false, 0, 5},
		{"exercise_500cal", "Calorie Crusher", "Burned 500+ calories in one session", "exercise", len(exercises) > 0 && exercises[0].CaloriesBurned >= 500, 0, 0},

		{"weight_first5", "First 5 Down", "Lost your first 5 kg", "weight", len(weights) >= 2, 0, 0},
		{"weight_10kg", "Double Digits", "Lost 10 kg total", "weight", false, 0, 0},
		{"weight_logged_7", "Consistent Scale", "Logged weight 7 days in a row", "weight", false, 0, 7},

		{"meals_10", "Getting Started", "Logged 10 meals total", "meals", false, 0, 10},
		{"meals_50", "Half Century", "Logged 50 meals total", "meals", false, 0, 50},
		{"meals_100", "Meal Master", "Logged 100 meals total", "meals", false, 0, 100},

		{"early_bird", "Early Bird", "Logged a meal before 7 AM", "meals", false, 0, 0},
		{"night_owl", "Night Owl", "Logged a meal after 10 PM", "meals", false, 0, 0},
	}

	for _, c := range checks {
		if c.Met {
			h.q.UnlockAchievement(ctx, sqlc.UnlockAchievementParams{
				UserID:          userID,
				Type:            c.Type,
				Title:           c.Title,
				Description:     c.Description,
				Category:        c.Category,
				ProgressCurrent: int32(c.ProgressCurrent),
				ProgressTarget:  int32(c.ProgressTarget),
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
			ID:              a.ID,
			Type:            a.Type,
			Title:           a.Title,
			Description:     a.Description,
			Category:        a.Category,
			UnlockedAt:      a.UnlockedAt.Format(time.RFC3339),
			ProgressCurrent: int(a.ProgressCurrent),
			ProgressTarget:  int(a.ProgressTarget),
		})
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}
