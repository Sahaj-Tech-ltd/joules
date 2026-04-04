package coach

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/admin"

	"joules/internal/ai"
	"joules/internal/auth"
	"joules/internal/config"
	"joules/internal/db/sqlc"
	"joules/internal/exercise"
	syslog "joules/internal/syslog"
)

type Handler struct {
	q    *sqlc.Queries
	ai   ai.Client
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewHandler(q *sqlc.Queries, aiClient ai.Client, pool *pgxpool.Pool, cfg *config.Config) *Handler {
	return &Handler{q: q, ai: aiClient, pool: pool, cfg: cfg}
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

func getCoachUserID(r *http.Request) (string, error) {
	userID, ok := r.Context().Value(auth.ContextUserID).(string)
	if !ok {
		return "", fmt.Errorf("unauthorized")
	}
	return userID, nil
}

func getUserID(r *http.Request) (string, error) {
	userID, ok := r.Context().Value(auth.ContextUserID).(string)
	if !ok {
		return "", fmt.Errorf("unauthorized")
	}
	return userID, nil
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

type tipsCacheEntry struct {
	tips     string
	cachedAt time.Time
}

var tipsCache sync.Map

func init() {
	go func() {
		for range time.Tick(4 * time.Hour) {
			now := time.Now()
			tipsCache.Range(func(key, value any) bool {
				if entry, ok := value.(tipsCacheEntry); ok && now.Sub(entry.cachedAt) > 4*time.Hour {
					tipsCache.Delete(key)
				}
				return true
			})
		}
	}()
}

type chatMessageRequest struct {
	Content string `json:"content"`
}

type chatMessageResponse struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func profileContext(profile sqlc.UserProfile, goals sqlc.UserGoal) string {
	age := int32(0)
	if profile.Age != nil {
		age = *profile.Age
	}
	sex := "unknown"
	if profile.Sex != nil {
		sex = *profile.Sex
	}
	activity := "unknown"
	if profile.ActivityLevel != nil {
		activity = *profile.ActivityLevel
	}

	return fmt.Sprintf(
		"User: %s | Age: %d | Sex: %s | Weight: %.1fkg | Height: %.1fcm | Activity: %s\nGoal: %s | Eating plan: %s | Targets: %d kcal/day, %dg protein, %dg carbs, %dg fat",
		profile.Name, age, sex,
		numericToFloat(profile.WeightKg),
		numericToFloat(profile.HeightCm),
		activity,
		goals.Objective, goals.DietPlan,
		goals.DailyCalorieTarget,
		goals.DailyProteinG, goals.DailyCarbsG, goals.DailyFatG,
	)
}

func (h *Handler) fetchPreferencesContext(ctx context.Context, userID string) string {
	var dietType, foodNotes, eatingContext string
	var allergies []string
	err := h.pool.QueryRow(ctx,
		"SELECT diet_type, allergies, food_notes, eating_context FROM user_preferences WHERE user_id = $1",
		userID,
	).Scan(&dietType, &allergies, &foodNotes, &eatingContext)
	if err != nil {
		return ""
	}
	var parts []string
	if dietType != "" && dietType != "omnivore" {
		parts = append(parts, "Diet type: "+dietType)
	}
	if len(allergies) > 0 {
		parts = append(parts, "Allergies/intolerances: "+strings.Join(allergies, ", "))
	}
	if foodNotes != "" {
		parts = append(parts, "Food preferences: "+foodNotes)
	}
	if eatingContext != "" {
		parts = append(parts, "Eating context: "+eatingContext)
	}
	if len(parts) == 0 {
		return ""
	}
	return "\nUser food preferences:\n" + strings.Join(parts, "\n")
}

// fetchStepsContext returns a line with today's step count for the system prompt.
func (h *Handler) fetchStepsContext(ctx context.Context, userID string) string {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	var steps int32
	err := h.pool.QueryRow(ctx,
		"SELECT step_count FROM step_logs WHERE user_id = $1 AND date = $2",
		userID, today,
	).Scan(&steps)
	if err != nil || steps == 0 {
		return ""
	}
	return fmt.Sprintf("\nSteps today: %d", steps)
}

// fetchWeightTrend returns a human-readable weight trend string based on the last 14 days.
func (h *Handler) fetchWeightTrend(ctx context.Context, userID string) string {
	now := time.Now()
	twoWeeksAgo := now.AddDate(0, 0, -14)
	logs, err := h.q.GetWeightHistory(ctx, sqlc.GetWeightHistoryParams{
		UserID: userID,
		Date:   twoWeeksAgo,
		Date_2: now,
	})
	if err != nil || len(logs) < 2 {
		return "stable (not enough data)"
	}
	first := numericToFloat(logs[0].WeightKg)
	last := numericToFloat(logs[len(logs)-1].WeightKg)
	diff := last - first
	if diff > 0.2 {
		return fmt.Sprintf("gained %.1f kg in last 14 days", diff)
	} else if diff < -0.2 {
		return fmt.Sprintf("lost %.1f kg in last 14 days", -diff)
	}
	return "stable"
}

// fetchLoggingStreak returns the number of consecutive days (ending today) where the user logged meals.
func (h *Handler) fetchLoggingStreak(ctx context.Context, userID string) int {
	rows, err := h.pool.Query(ctx,
		`SELECT DISTINCT DATE(timestamp AT TIME ZONE 'UTC') AS day
		 FROM meals
		 WHERE user_id = $1 AND timestamp >= NOW() - INTERVAL '90 days'
		 ORDER BY day DESC`,
		userID,
	)
	if err != nil {
		return 0
	}
	defer rows.Close()

	streak := 0
	expected := time.Now().UTC().Truncate(24 * time.Hour)
	for rows.Next() {
		var day time.Time
		if err := rows.Scan(&day); err != nil {
			break
		}
		dayUTC := day.UTC().Truncate(24 * time.Hour)
		if dayUTC.Equal(expected) {
			streak++
			expected = expected.AddDate(0, 0, -1)
		} else if streak == 0 && dayUTC.Equal(expected.AddDate(0, 0, -1)) {
			// allow streak starting from yesterday if nothing logged today yet
			streak++
			expected = dayUTC.AddDate(0, 0, -1)
		} else {
			break
		}
	}
	return streak
}

// fetchRecentChatContext returns a condensed summary of the last few user messages.
func (h *Handler) fetchRecentChatContext(ctx context.Context, userID string) (string, time.Time) {
	rows, err := h.pool.Query(ctx,
		`SELECT role, content, created_at FROM coach_messages
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT 5`,
		userID,
	)
	if err != nil {
		return "", time.Time{}
	}
	defer rows.Close()

	type msg struct {
		role      string
		content   string
		createdAt time.Time
	}
	var messages []msg
	var latestTime time.Time
	for rows.Next() {
		var m msg
		if err := rows.Scan(&m.role, &m.content, &m.createdAt); err != nil {
			continue
		}
		messages = append(messages, m)
		if m.createdAt.After(latestTime) {
			latestTime = m.createdAt
		}
	}

	if len(messages) == 0 {
		return "", time.Time{}
	}

	// Collect up to 3 user messages in chronological order (messages are DESC, so reverse)
	var userMsgs []string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].role == "user" {
			content := messages[i].content
			if len(content) > 100 {
				content = content[:100] + "…"
			}
			userMsgs = append(userMsgs, content)
		}
		if len(userMsgs) >= 3 {
			break
		}
	}

	if len(userMsgs) == 0 {
		return "", latestTime
	}
	return strings.Join(userMsgs, "; "), latestTime
}

func defaultTips(profile sqlc.UserProfile, goals sqlc.UserGoal) string {
	firstName := profile.Name
	if firstName == "" {
		firstName = "there"
	}
	if i := strings.Index(firstName, " "); i > 0 {
		firstName = firstName[:i]
	}

	tips := []string{
		fmt.Sprintf("Welcome, %s! I'm Joules, your AI nutrition coach. Start by logging your first meal to get personalized advice.", firstName),
	}

	switch goals.DietPlan {
	case "keto":
		tips = append(tips, "On keto, aim to keep carbs under 25g/day — focus on healthy fats like avocado, olive oil, and nuts.")
	case "intermittent_fasting":
		tips = append(tips, "Stay hydrated during your fasting window with water, black coffee, or tea to manage hunger and maintain energy.")
	case "paleo":
		tips = append(tips, "Paleo means whole, unprocessed foods — prioritize lean meats, vegetables, fruits, and nuts over packaged foods.")
	case "mediterranean":
		tips = append(tips, "The Mediterranean diet shines with olive oil, fish, legumes, and colorful vegetables — great for heart health and satiety.")
	default:
		if goals.DailyProteinG > 0 {
			tips = append(tips, fmt.Sprintf("Aim for %dg of protein daily — spread it across meals with sources like eggs, chicken, legumes, or Greek yogurt.", goals.DailyProteinG))
		} else {
			tips = append(tips, "Aim to include a quality protein source at every meal to support energy levels and reduce cravings.")
		}
	}

	switch goals.Objective {
	case "cut_fat":
		tips = append(tips, "To lose fat, a consistent calorie deficit matters more than any single food choice — log everything honestly.")
	case "build_muscle":
		tips = append(tips, "For muscle growth, prioritize hitting your protein goal every day and aim to eat at a small calorie surplus.")
	case "feel_better":
		tips = append(tips, "Eating enough vegetables, staying hydrated, and getting consistent sleep will make the biggest difference in how you feel.")
	default:
		tips = append(tips, "Consistency beats perfection — even tracking 80% of your meals gives you powerful data to improve.")
	}

	tips = append(tips, "Head to the coach chat anytime to ask about nutrition, get meal ideas, or log food and water with a simple message!")
	return strings.Join(tips, "\n")
}

func (h *Handler) GetTips(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	today := time.Now().Format("2006-01-02")
	cacheKey := userID + ":" + today

	ctx := r.Context()

	// Check cache, but bust if there are coach messages newer than cache time
	if cached, ok := tipsCache.Load(cacheKey); ok {
		entry := cached.(tipsCacheEntry)
		// Check if any coach messages today exist (regenerate if so)
		var msgCount int
		_ = h.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM coach_messages WHERE user_id = $1 AND created_at > $2`,
			userID, entry.cachedAt,
		).Scan(&msgCount)
		if msgCount == 0 {
			writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"tips": entry.tips}})
			return
		}
	}

	profile, err := h.q.GetProfile(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get profile: %w", err))
		return
	}

	goals, err := h.q.GetGoals(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get goals: %w", err))
		return
	}

	// If no AI client configured, return personalized static tips
	if h.ai == nil {
		tips := defaultTips(profile, goals)
		tipsCache.Store(cacheKey, tipsCacheEntry{tips: tips, cachedAt: time.Now()})
		writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"tips": tips}})
		return
	}

	summary, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{
		UserID:    userID,
		Timestamp: time.Now(),
		Column3:   "UTC",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get daily summary: %w", err))
		return
	}

	weightTrend := h.fetchWeightTrend(ctx, userID)
	streak := h.fetchLoggingStreak(ctx, userID)
	chatContext, _ := h.fetchRecentChatContext(ctx, userID)

	userName := profile.Name
	if userName == "" {
		userName = "there"
	}

	streakStr := fmt.Sprintf("%d consecutive day(s) of logging", streak)
	if streak == 0 {
		streakStr = "no recent logging streak"
	}

	contextSection := fmt.Sprintf(
		"\nWeight trend: %s\nLogging streak: %s",
		weightTrend, streakStr,
	)
	if chatContext != "" {
		contextSection = fmt.Sprintf(
			"\nRecent context from chat: %s\nWeight trend: %s\nLogging streak: %s",
			chatContext, weightTrend, streakStr,
		)
	}

	tipsPrompt := admin.GetSettingDefault(h.pool, ctx, "prompt_tips", admin.DefaultPrompts["prompt_tips"])
	prompt := fmt.Sprintf(
		tipsPrompt,
		userName,
		profileContext(profile, goals),
		summary.TotalCalories,
		goals.DailyCalorieTarget,
		summary.TotalProtein,
		goals.DailyProteinG,
		summary.TotalWaterMl,
		contextSection,
	)
	prompt += h.fetchPreferencesContext(ctx, userID)

	tips, err := h.ai.Chat(prompt, nil)
	if err != nil {
		// AI failed — fall back to static personalized tips rather than returning an error
		syslog.Warn("ai", "Tips generation failed, using fallback", map[string]any{"user_id": userID, "error": err.Error()})
		tips = defaultTips(profile, goals)
	}

	tipsCache.Store(cacheKey, tipsCacheEntry{tips: tips, cachedAt: time.Now()})
	syslog.Info("ai", "Daily tips generated", map[string]any{"user_id": userID, "date": today})

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"tips": tips}})
}

func (h *Handler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	messages, err := h.q.GetCoachHistory(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get chat history: %w", err))
		return
	}

	resp := make([]chatMessageResponse, 0, len(messages))
	for _, m := range messages {
		resp = append(resp, chatMessageResponse{
			ID:        m.ID,
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

// agentTools returns the list of tools available to the coach agent.
// If TavilyAPIKey is set in cfg, the web search tool is included.
func (h *Handler) agentTools() []ai.Tool {
	tools := []ai.Tool{
		searchHistoryTool(),
		{
			Name:        "log_water",
			Description: "Log water intake for the user. Use this when the user says they drank water or any beverage.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"amount_ml": map[string]interface{}{
						"type":        "integer",
						"description": "Amount of water in millilitres (e.g. 250 for a glass, 500 for a bottle)",
					},
				},
				"required": []string{"amount_ml"},
			},
		},
		{
			Name:        "log_exercise",
			Description: "Log an exercise session for the user. Calories burned are auto-calculated based on the exercise type (MET value) and user's weight. Use this when the user mentions working out, going for a run, doing yoga, etc.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the exercise (e.g. 'Running', 'Yoga', 'Weight Training')",
					},
					"duration_min": map[string]interface{}{
						"type":        "integer",
						"description": "Duration of the exercise in minutes",
					},
				},
				"required": []string{"name", "duration_min"},
			},
		},
		{
			Name:        "log_food",
			Description: "Log a meal or food item for the user. Use this when the user tells you what they ate or asks you to log food. Estimate macros from typical nutritional values if not provided.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the food or meal",
					},
					"calories": map[string]interface{}{
						"type":        "integer",
						"description": "Estimated calories",
					},
					"protein_g": map[string]interface{}{
						"type":        "number",
						"description": "Protein in grams",
					},
					"carbs_g": map[string]interface{}{
						"type":        "number",
						"description": "Carbohydrates in grams",
					},
					"fat_g": map[string]interface{}{
						"type":        "number",
						"description": "Fat in grams",
					},
					"meal_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"breakfast", "lunch", "dinner", "snack"},
						"description": "Type of meal",
					},
				},
				"required": []string{"name", "calories", "protein_g", "carbs_g", "fat_g", "meal_type"},
			},
		},
		{
			Name:        "get_today_summary",
			Description: "Get today's nutrition and activity summary for the user. Use this when the user asks about their progress, stats, or how they're doing today.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "create_achievement",
			Description: "Create a custom achievement/badge to celebrate a user milestone or accomplishment. Use this to reward the user for good habits or hitting goals.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Short title for the achievement (e.g. 'Hydration Hero')",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Brief description of what the achievement is for",
					},
				},
				"required": []string{"title", "description"},
			},
		},
		{
			Name:        "get_fasting_context",
			Description: "Get the user's current intermittent fasting status and context. Use this when the user asks what to eat to break their fast, asks for break-fast meal suggestions, or asks how their fast is going. Returns fasting duration, protocol, eating window, calorie targets, and dietary restrictions.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "log_weight",
			Description: "Log the user's current body weight. Use this when the user tells you their weight.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"weight_kg": map[string]interface{}{
						"type":        "number",
						"description": "Weight in kilograms",
					},
				},
				"required": []string{"weight_kg"},
			},
		},
		{
			Name:        "log_steps",
			Description: "Log the user's step count for today. Use this when the user mentions how many steps they walked.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"step_count": map[string]interface{}{
						"type":        "integer",
						"description": "Number of steps",
					},
				},
				"required": []string{"step_count"},
			},
		},
		{
			Name:        "update_daily_tips",
			Description: "Update the user's daily tips section with new personalized tips. Use this when the user asks you to update or change their tips, or when you want to set fresh tips based on new context.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tips": map[string]interface{}{
						"type":        "string",
						"description": "New tips content to display (bullet points, concise, actionable)",
					},
				},
				"required": []string{"tips"},
			},
		},
		{
			Name:        "save_memory",
			Description: "Save a fact about the user to long-term memory. Use this when you learn about allergies, preferences, habits, routines, health conditions, or goals that should be remembered across sessions.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"allergy", "preference", "habit", "routine", "goal", "health_condition", "misc"},
						"description": "Category of the memory",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The fact to remember",
					},
				},
				"required": []string{"category", "content"},
			},
		},
		{
			Name:        "search_memory",
			Description: "Search the user's long-term memory for facts previously saved. Use this when you need to recall preferences, allergies, habits, or other context.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "update_goals",
			Description: "Update the user's nutrition targets. All parameters are optional — only provided fields are updated. Minimum calorie target is 1200.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"daily_calorie_target": map[string]interface{}{"type": "integer", "description": "New daily calorie target"},
					"daily_protein_g":      map[string]interface{}{"type": "integer", "description": "New daily protein target in grams"},
					"daily_carbs_g":        map[string]interface{}{"type": "integer", "description": "New daily carbs target in grams"},
					"daily_fat_g":          map[string]interface{}{"type": "integer", "description": "New daily fat target in grams"},
				},
			},
		},
		{
			Name:        "update_profile",
			Description: "Update the user's profile fields. Always confirm with the user before making changes.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":             map[string]interface{}{"type": "string", "description": "Display name"},
					"age":              map[string]interface{}{"type": "integer", "description": "Age in years"},
					"activity_level":   map[string]interface{}{"type": "string", "enum": []string{"sedentary", "light", "moderate", "active", "very_active"}, "description": "Activity level"},
					"target_weight_kg": map[string]interface{}{"type": "number", "description": "Target weight in kg"},
				},
			},
		},
		{
			Name:        "get_progress_report",
			Description: "Get an aggregated progress report for a time period. Returns average daily macros, weight change, logging stats, and comparison to targets.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"period": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"week", "month", "3months"},
						"description": "Time range for the report",
					},
				},
				"required": []string{"period"},
			},
		},
		{
			Name:        "suggest_meal_plan",
			Description: "Get remaining macro budget and context for meal suggestions. Returns today's consumed vs target macros so you can suggest appropriate meals.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"meal_type":      map[string]interface{}{"type": "string", "enum": []string{"breakfast", "lunch", "dinner", "snack"}, "description": "Which meal to plan for"},
					"calorie_budget": map[string]interface{}{"type": "integer", "description": "Optional calorie budget override"},
				},
				"required": []string{"meal_type"},
			},
		},
		{
			Name:        "set_reminder",
			Description: "Schedule a reminder for the user. The reminder is saved and will be shown in their notifications.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type":    map[string]interface{}{"type": "string", "enum": []string{"meal", "water", "fasting", "custom"}, "description": "Type of reminder"},
					"message": map[string]interface{}{"type": "string", "description": "Reminder message"},
					"time":    map[string]interface{}{"type": "string", "description": "Time in HH:MM format (24h)"},
				},
				"required": []string{"type", "message", "time"},
			},
		},
	}

	if h.cfg != nil && h.cfg.TavilyAPIKey != "" {
		tools = append(tools, ai.Tool{
			Name:        "search_web",
			Description: "Search the web for current nutrition, fitness, or health information. Use this only when you need up-to-date facts you're uncertain about.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
				},
				"required": []string{"query"},
			},
		})
	}

	tools = append(tools, ai.Tool{
		Name:        "fetch_url",
		Description: "Fetch the text content of a URL. Use this to retrieve nutrition info from restaurant websites, food brand pages, or health articles. Only use with http/https URLs.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The URL to fetch (must start with http:// or https://)",
				},
			},
			"required": []string{"url"},
		},
	})

	tools = append(tools, ai.Tool{
		Name:        "lookup_nutrition",
		Description: "Look up nutrition information for a specific food or menu item. Checks a persistent cache first, then searches the web. Results are stored globally for future queries.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"food_name": map[string]interface{}{
					"type":        "string",
					"description": "The food or menu item to look up (e.g. 'Taco Bell Crunchy Taco', 'Big Mac', 'Chipotle Chicken Bowl')",
				},
			},
			"required": []string{"food_name"},
		},
	})

	return tools
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n])
	}
	return s
}

// executeTool runs a single tool call and returns the result as a string.
func (h *Handler) executeTool(ctx context.Context, userID string, toolName string, argsJSON string) string {
	slog.Info("agent executing tool", "tool", toolName, "args", argsJSON)

	switch toolName {
	case "search_my_history":
		return h.executeSearchHistory(ctx, userID, argsJSON)

	case "log_water":
		var args struct {
			AmountMl int `json:"amount_ml"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.AmountMl <= 0 {
			return "error: amount_ml must be positive"
		}
		_, err := h.q.LogWater(ctx, sqlc.LogWaterParams{
			UserID:   userID,
			Date:     time.Now(),
			AmountMl: int32(args.AmountMl),
		})
		if err != nil {
			slog.Error("log_water tool failed", "error", err)
			return fmt.Sprintf("error logging water: %v", err)
		}
		syslog.Info("ai", "Agent tool executed", map[string]any{"user_id": userID, "tool": "log_water", "args_summary": truncate(argsJSON, 100)})
		return fmt.Sprintf("Successfully logged %dml of water.", args.AmountMl)

	case "log_exercise":
		var args struct {
			Name        string `json:"name"`
			DurationMin int    `json:"duration_min"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.Name == "" {
			return "error: exercise name is required"
		}
		if args.DurationMin <= 0 {
			return "error: duration_min must be positive"
		}
		var profileWeight float64
		p, err := h.q.GetProfile(ctx, userID)
		if err == nil {
			profileWeight = numericToFloat(p.WeightKg)
		}
		if profileWeight <= 0 {
			profileWeight = 70.0
		}
		met := exercise.FindMET(args.Name)
		calories := exercise.CalculateCalories(met, profileWeight, int32(args.DurationMin))
		_, err = h.q.LogExercise(ctx, sqlc.LogExerciseParams{
			UserID:         userID,
			Timestamp:      time.Now(),
			Name:           args.Name,
			DurationMin:    int32(args.DurationMin),
			CaloriesBurned: calories,
		})
		if err != nil {
			slog.Error("log_exercise tool failed", "error", err)
			return fmt.Sprintf("error logging exercise: %v", err)
		}
		syslog.Info("ai", "Agent tool executed", map[string]any{"user_id": userID, "tool": "log_exercise", "args_summary": truncate(argsJSON, 100)})
		return fmt.Sprintf("Successfully logged %s: %d minutes, %d calories burned.", args.Name, args.DurationMin, calories)

	case "log_food":
		var args struct {
			Name     string  `json:"name"`
			Calories int     `json:"calories"`
			ProteinG float64 `json:"protein_g"`
			CarbsG   float64 `json:"carbs_g"`
			FatG     float64 `json:"fat_g"`
			MealType string  `json:"meal_type"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.Name == "" {
			return "error: food name is required"
		}
		mealType := args.MealType
		if mealType == "" {
			mealType = "snack"
		}

		meal, err := h.q.CreateMeal(ctx, sqlc.CreateMealParams{
			UserID:    userID,
			Timestamp: time.Now(),
			MealType:  mealType,
			PhotoPath: nil,
			Note:      nil,
		})
		if err != nil {
			slog.Error("log_food create meal failed", "error", err)
			return fmt.Sprintf("error creating meal: %v", err)
		}

		proteinNumeric := pgtype.Numeric{}
		_ = proteinNumeric.Scan(fmt.Sprintf("%.2f", args.ProteinG))
		carbsNumeric := pgtype.Numeric{}
		_ = carbsNumeric.Scan(fmt.Sprintf("%.2f", args.CarbsG))
		fatNumeric := pgtype.Numeric{}
		_ = fatNumeric.Scan(fmt.Sprintf("%.2f", args.FatG))
		fiberNumeric := pgtype.Numeric{}
		_ = fiberNumeric.Scan("0")

		_, err = h.q.CreateFoodItem(ctx, sqlc.CreateFoodItemParams{
			MealID:      meal.ID,
			Name:        args.Name,
			Calories:    int32(args.Calories),
			ProteinG:    proteinNumeric,
			CarbsG:      carbsNumeric,
			FatG:        fatNumeric,
			FiberG:      fiberNumeric,
			ServingSize: nil,
			Source:      "ai",
		})
		if err != nil {
			slog.Error("log_food create food item failed", "error", err)
			return fmt.Sprintf("error creating food item: %v", err)
		}
		syslog.Info("ai", "Agent tool executed", map[string]any{"user_id": userID, "tool": "log_food", "args_summary": truncate(argsJSON, 100)})
		return fmt.Sprintf("Successfully logged %s (%d kcal, %.0fg protein, %.0fg carbs, %.0fg fat) as %s.",
			args.Name, args.Calories, args.ProteinG, args.CarbsG, args.FatG, mealType)

	case "get_today_summary":
		summary, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{
			UserID:    userID,
			Timestamp: time.Now(),
			Column3:   "UTC",
		})
		if err != nil {
			slog.Error("get_today_summary failed", "error", err)
			return fmt.Sprintf("error fetching summary: %v", err)
		}
		goals, err := h.q.GetGoals(ctx, userID)
		if err != nil {
			return fmt.Sprintf(`{"calories":%d,"protein_g":%.1f,"carbs_g":%.1f,"fat_g":%.1f,"fiber_g":%.1f,"calories_burned":%d,"water_ml":%d}`,
				summary.TotalCalories, summary.TotalProtein, summary.TotalCarbs, summary.TotalFat, summary.TotalFiber, summary.TotalBurned, summary.TotalWaterMl)
		}
		return fmt.Sprintf(`{"calories":%d,"calorie_target":%d,"protein_g":%.1f,"protein_target_g":%d,"carbs_g":%.1f,"carbs_target_g":%d,"fat_g":%.1f,"fat_target_g":%d,"fiber_g":%.1f,"calories_burned":%d,"water_ml":%d}`,
			summary.TotalCalories, goals.DailyCalorieTarget,
			summary.TotalProtein, goals.DailyProteinG,
			summary.TotalCarbs, goals.DailyCarbsG,
			summary.TotalFat, goals.DailyFatG,
			summary.TotalFiber,
			summary.TotalBurned,
			summary.TotalWaterMl)

	case "create_achievement":
		var args struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.Title == "" {
			return "error: achievement title is required"
		}
		// Derive a unique type string from the title
		achievementType := "custom_" + strings.ToLower(strings.ReplaceAll(args.Title, " ", "_"))
		_, err := h.q.UnlockAchievement(ctx, sqlc.UnlockAchievementParams{
			UserID:      userID,
			Type:        achievementType,
			Title:       args.Title,
			Description: args.Description,
		})
		if err != nil {
			slog.Error("create_achievement failed", "error", err)
			return fmt.Sprintf("error creating achievement: %v", err)
		}
		syslog.Info("ai", "Agent tool executed", map[string]any{"user_id": userID, "tool": "create_achievement", "args_summary": truncate(argsJSON, 100)})
		return fmt.Sprintf("Achievement unlocked: '%s' — %s", args.Title, args.Description)

	case "search_web":
		if h.cfg == nil || h.cfg.TavilyAPIKey == "" {
			return "error: web search is not configured"
		}
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.Query == "" {
			return "error: query is required"
		}
		result, err := ai.SearchWeb(h.cfg.TavilyAPIKey, args.Query)
		if err != nil {
			slog.Error("search_web tool failed", "error", err)
			return fmt.Sprintf("error searching web: %v", err)
		}
		syslog.Info("ai", "Agent tool executed", map[string]any{"user_id": userID, "tool": "search_web", "args_summary": truncate(argsJSON, 100)})
		return result

	case "fetch_url":
		var args struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if !strings.HasPrefix(args.URL, "http://") && !strings.HasPrefix(args.URL, "https://") {
			return "error: URL must start with http:// or https://"
		}
		text, err := h.fetchURL(args.URL)
		if err != nil {
			slog.Error("fetch_url tool failed", "error", err)
			return fmt.Sprintf("error fetching URL: %v", err)
		}
		return text

	case "lookup_nutrition":
		var args struct {
			FoodName string `json:"food_name"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.FoodName == "" {
			return "error: food_name is required"
		}
		result, err := h.lookupNutrition(ctx, args.FoodName)
		if err != nil {
			slog.Error("lookup_nutrition tool failed", "error", err)
			return fmt.Sprintf("error looking up nutrition: %v", err)
		}
		syslog.Info("ai", "Agent tool executed", map[string]any{"user_id": userID, "tool": "lookup_nutrition", "args_summary": truncate(argsJSON, 100)})
		return result

	case "get_fasting_context":
		return h.executeFastingContext(ctx, userID)

	case "log_weight":
		var args struct {
			WeightKg float64 `json:"weight_kg"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.WeightKg <= 0 || args.WeightKg > 500 {
			return "error: weight_kg must be a reasonable value between 0 and 500"
		}
		weightNumeric := pgtype.Numeric{}
		_ = weightNumeric.Scan(fmt.Sprintf("%.1f", args.WeightKg))
		_, err := h.q.LogWeight(ctx, sqlc.LogWeightParams{
			UserID:   userID,
			Date:     time.Now(),
			WeightKg: weightNumeric,
		})
		if err != nil {
			slog.Error("log_weight tool failed", "error", err)
			return fmt.Sprintf("error logging weight: %v", err)
		}
		syslog.Info("ai", "Agent tool executed", map[string]any{"user_id": userID, "tool": "log_weight", "args_summary": truncate(argsJSON, 100)})
		return fmt.Sprintf("Successfully logged weight: %.1f kg.", args.WeightKg)

	case "log_steps":
		var args struct {
			StepCount int `json:"step_count"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.StepCount <= 0 {
			return "error: step_count must be positive"
		}
		_, err := h.q.LogSteps(ctx, sqlc.LogStepsParams{
			UserID:    userID,
			Date:      time.Now(),
			StepCount: int32(args.StepCount),
			Source:    "coach",
		})
		if err != nil {
			slog.Error("log_steps tool failed", "error", err)
			return fmt.Sprintf("error logging steps: %v", err)
		}
		syslog.Info("ai", "Agent tool executed", map[string]any{"user_id": userID, "tool": "log_steps", "args_summary": truncate(argsJSON, 100)})
		return fmt.Sprintf("Successfully logged %d steps.", args.StepCount)

	case "update_daily_tips":
		var args struct {
			Tips string `json:"tips"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.Tips == "" {
			return "error: tips content is required"
		}
		today := time.Now().Format("2006-01-02")
		cacheKey := userID + ":" + today
		tipsCache.Store(cacheKey, tipsCacheEntry{tips: args.Tips, cachedAt: time.Now()})
		syslog.Info("ai", "Agent tool executed", map[string]any{"user_id": userID, "tool": "update_daily_tips"})
		return "Daily tips updated successfully."

	case "save_memory":
		var args struct {
			Category string `json:"category"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		validCategories := map[string]bool{"allergy": true, "preference": true, "habit": true, "routine": true, "goal": true, "health_condition": true, "misc": true}
		if !validCategories[args.Category] {
			return "error: invalid category"
		}
		if args.Content == "" {
			return "error: content is required"
		}
		err := SaveMemory(ctx, h.pool, userID, args.Category, args.Content, "agent")
		if err != nil {
			return fmt.Sprintf("error saving memory: %v", err)
		}
		return fmt.Sprintf("Remembered: [%s] %s", args.Category, args.Content)

	case "search_memory":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		results, err := SearchMemory(ctx, h.pool, userID, args.Query)
		if err != nil {
			return fmt.Sprintf("error searching memory: %v", err)
		}
		if len(results) == 0 {
			return "No matching memories found."
		}
		var sb strings.Builder
		for _, m := range results {
			sb.WriteString(fmt.Sprintf("- [%s] %s (saved %s)\n", m.Category, m.Content, m.CreatedAt.Format("Jan 2")))
		}
		return sb.String()

	case "update_goals":
		var args struct {
			DailyCalorieTarget *int `json:"daily_calorie_target"`
			DailyProteinG      *int `json:"daily_protein_g"`
			DailyCarbsG        *int `json:"daily_carbs_g"`
			DailyFatG          *int `json:"daily_fat_g"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.DailyCalorieTarget != nil && *args.DailyCalorieTarget < 1200 {
			return "error: calorie target must be at least 1200"
		}
		var setClauses []string
		var setArgs []interface{}
		argIdx := 2
		if args.DailyCalorieTarget != nil {
			setClauses = append(setClauses, fmt.Sprintf("daily_calorie_target = $%d", argIdx))
			setArgs = append(setArgs, *args.DailyCalorieTarget)
			argIdx++
		}
		if args.DailyProteinG != nil {
			setClauses = append(setClauses, fmt.Sprintf("daily_protein_g = $%d", argIdx))
			setArgs = append(setArgs, *args.DailyProteinG)
			argIdx++
		}
		if args.DailyCarbsG != nil {
			setClauses = append(setClauses, fmt.Sprintf("daily_carbs_g = $%d", argIdx))
			setArgs = append(setArgs, *args.DailyCarbsG)
			argIdx++
		}
		if args.DailyFatG != nil {
			setClauses = append(setClauses, fmt.Sprintf("daily_fat_g = $%d", argIdx))
			setArgs = append(setArgs, *args.DailyFatG)
			argIdx++
		}
		if len(setClauses) == 0 {
			return "error: no fields provided to update"
		}
		query := "UPDATE user_goals SET " + strings.Join(setClauses, ", ") + " WHERE user_id = $1"
		allArgs := append([]interface{}{userID}, setArgs...)
		_, err := h.pool.Exec(ctx, query, allArgs...)
		if err != nil {
			return fmt.Sprintf("error updating goals: %v", err)
		}
		return "Goals updated successfully."

	case "update_profile":
		var args struct {
			Name           *string  `json:"name"`
			Age            *int     `json:"age"`
			ActivityLevel  *string  `json:"activity_level"`
			TargetWeightKg *float64 `json:"target_weight_kg"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		var setClauses []string
		var setArgs []interface{}
		argIdx := 2
		if args.Name != nil {
			setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
			setArgs = append(setArgs, *args.Name)
			argIdx++
		}
		if args.Age != nil {
			setClauses = append(setClauses, fmt.Sprintf("age = $%d", argIdx))
			setArgs = append(setArgs, *args.Age)
			argIdx++
		}
		if args.ActivityLevel != nil {
			validLevels := map[string]bool{"sedentary": true, "light": true, "moderate": true, "active": true, "very_active": true}
			if !validLevels[*args.ActivityLevel] {
				return "error: invalid activity_level"
			}
			setClauses = append(setClauses, fmt.Sprintf("activity_level = $%d", argIdx))
			setArgs = append(setArgs, *args.ActivityLevel)
			argIdx++
		}
		if args.TargetWeightKg != nil {
			setClauses = append(setClauses, fmt.Sprintf("target_weight_kg = $%d", argIdx))
			setArgs = append(setArgs, *args.TargetWeightKg)
			argIdx++
		}
		if len(setClauses) == 0 {
			return "error: no fields provided to update"
		}
		query := "UPDATE user_profiles SET " + strings.Join(setClauses, ", ") + " WHERE user_id = $1"
		allArgs := append([]interface{}{userID}, setArgs...)
		_, err := h.pool.Exec(ctx, query, allArgs...)
		if err != nil {
			return fmt.Sprintf("error updating profile: %v", err)
		}
		return "Profile updated successfully."

	case "get_progress_report":
		var args struct {
			Period string `json:"period"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		days := 7
		switch args.Period {
		case "week":
			days = 7
		case "month":
			days = 30
		case "3months":
			days = 90
		default:
			return "error: period must be week, month, or 3months"
		}
		startDate := time.Now().AddDate(0, 0, -days)

		var avgCal, avgProtein, avgCarbs, avgFat float64
		var mealCount int
		h.pool.QueryRow(ctx,
			`SELECT COALESCE(AVG(total_cal), 0), COALESCE(AVG(total_protein), 0), COALESCE(AVG(total_carbs), 0), COALESCE(AVG(total_fat), 0), COALESCE(SUM(meal_count), 0)
			 FROM (
			    SELECT DATE(m.timestamp) as day,
			           SUM(fi.calories) as total_cal,
			           SUM(fi.protein_g)::float as total_protein,
			           SUM(fi.carbs_g)::float as total_carbs,
			           SUM(fi.fat_g)::float as total_fat,
			           COUNT(DISTINCT m.id) as meal_count
			    FROM meals m JOIN food_items fi ON fi.meal_id = m.id
			    WHERE m.user_id = $1 AND m.timestamp >= $2
			    GROUP BY DATE(m.timestamp)
			 ) sub`,
			userID, startDate,
		).Scan(&avgCal, &avgProtein, &avgCarbs, &avgFat, &mealCount)

		var firstWeight, lastWeight float64
		h.pool.QueryRow(ctx,
			`SELECT (SELECT weight_kg FROM weight_logs WHERE user_id = $1 AND date >= $2 ORDER BY date ASC LIMIT 1),
			        (SELECT weight_kg FROM weight_logs WHERE user_id = $1 AND date >= $2 ORDER BY date DESC LIMIT 1)`,
			userID, startDate,
		).Scan(&firstWeight, &lastWeight)

		var exerciseCount int
		h.pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM exercises WHERE user_id = $1 AND timestamp >= $2",
			userID, startDate,
		).Scan(&exerciseCount)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Progress report (last %d days):\n", days))
		sb.WriteString(fmt.Sprintf("Avg daily: %.0f cal, %.0fg protein, %.0fg carbs, %.0fg fat\n", avgCal, avgProtein, avgCarbs, avgFat))
		if firstWeight > 0 && lastWeight > 0 {
			diff := lastWeight - firstWeight
			sb.WriteString(fmt.Sprintf("Weight change: %.1f kg → %.1f kg (%+.1f kg)\n", firstWeight, lastWeight, diff))
		}
		sb.WriteString(fmt.Sprintf("Meals logged: %d | Exercises logged: %d\n", mealCount, exerciseCount))
		return sb.String()

	case "suggest_meal_plan":
		var args struct {
			MealType      string `json:"meal_type"`
			CalorieBudget *int   `json:"calorie_budget"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		summary, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{
			UserID:    userID,
			Timestamp: time.Now(),
			Column3:   "UTC",
		})
		if err != nil {
			summary.TotalCalories = 0
			summary.TotalProtein = 0
			summary.TotalCarbs = 0
			summary.TotalFat = 0
		}
		goals, err := h.q.GetGoals(ctx, userID)
		if err != nil {
			return "error: could not retrieve goals"
		}
		remCal := float64(goals.DailyCalorieTarget) - float64(summary.TotalCalories)
		remProtein := float64(goals.DailyProteinG) - float64(summary.TotalProtein)
		remCarbs := float64(goals.DailyCarbsG) - float64(summary.TotalCarbs)
		remFat := float64(goals.DailyFatG) - float64(summary.TotalFat)
		if args.CalorieBudget != nil {
			remCal = float64(*args.CalorieBudget)
		}

		var dietType string
		var allergies []string
		h.pool.QueryRow(ctx,
			"SELECT diet_type, allergies FROM user_preferences WHERE user_id = $1", userID,
		).Scan(&dietType, &allergies)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Meal suggestion request for: %s\n", args.MealType))
		sb.WriteString(fmt.Sprintf("Remaining today: %.0f cal, %.0fg protein, %.0fg carbs, %.0fg fat\n", remCal, remProtein, remCarbs, remFat))
		if dietType != "" && dietType != "omnivore" {
			sb.WriteString(fmt.Sprintf("Diet type: %s\n", dietType))
		}
		if len(allergies) > 0 {
			sb.WriteString(fmt.Sprintf("Allergies: %s\n", strings.Join(allergies, ", ")))
		}
		return sb.String()

	case "set_reminder":
		var args struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Time    string `json:"time"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		validTypes := map[string]bool{"meal": true, "water": true, "fasting": true, "custom": true}
		if !validTypes[args.Type] {
			return "error: invalid reminder type"
		}
		if args.Message == "" {
			return "error: message is required"
		}
		if len(args.Time) != 5 || args.Time[2] != ':' {
			return "error: time must be in HH:MM format"
		}
		var id string
		err := h.pool.QueryRow(ctx,
			"INSERT INTO coach_reminders (user_id, type, message, reminder_time) VALUES ($1, $2, $3, $4) RETURNING id",
			userID, args.Type, args.Message, args.Time,
		).Scan(&id)
		if err != nil {
			return fmt.Sprintf("error creating reminder: %v", err)
		}
		return fmt.Sprintf("Reminder set: %s at %s — %s", args.Type, args.Time, args.Message)

	default:
		return fmt.Sprintf("error: unknown tool '%s'", toolName)
	}
}

func (h *Handler) executeFastingContext(ctx context.Context, userID string) string {
	goals, err := h.q.GetGoals(ctx, userID)
	if err != nil {
		return "error: could not retrieve fasting status"
	}

	if goals.DietPlan != "intermittent_fasting" {
		return "User is not on an intermittent fasting plan."
	}

	fastingWindow := "16:8"
	if goals.FastingWindow != nil {
		fastingWindow = *goals.FastingWindow
	}
	eatingHours := 8
	switch fastingWindow {
	case "18:6":
		eatingHours = 6
	case "20:4":
		eatingHours = 4
	case "omad":
		eatingHours = 1
	}
	fastingHours := 24 - eatingHours

	windowStart := "12:00"
	if goals.EatingWindowStart.Valid {
		h := goals.EatingWindowStart.Microseconds / 3600_000_000
		m := (goals.EatingWindowStart.Microseconds % 3600_000_000) / 60_000_000
		windowStart = fmt.Sprintf("%02d:%02d", h, m)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Fasting protocol: %s (%dh fast / %dh eating window)\n", fastingWindow, fastingHours, eatingHours))
	sb.WriteString(fmt.Sprintf("Eating window starts: %s\n", windowStart))
	sb.WriteString(fmt.Sprintf("Daily calorie target: %d kcal\n", goals.DailyCalorieTarget))
	sb.WriteString(fmt.Sprintf("Macro targets: %dg protein, %dg carbs, %dg fat\n", goals.DailyProteinG, goals.DailyCarbsG, goals.DailyFatG))

	streak := int32(0)
	if goals.FastingStreak != nil {
		streak = *goals.FastingStreak
	}

	if goals.CurrentFastStart.Valid {
		elapsed := time.Since(goals.CurrentFastStart.Time)
		hoursElapsed := int(elapsed.Hours())
		minutesElapsed := int(elapsed.Minutes()) % 60
		sb.WriteString(fmt.Sprintf("Currently fasting: YES — %dh %dm elapsed\n", hoursElapsed, minutesElapsed))
		remaining := time.Duration(fastingHours)*time.Hour - elapsed
		if remaining > 0 {
			sb.WriteString(fmt.Sprintf("Time remaining in fast: %dh %dm\n", int(remaining.Hours()), int(remaining.Minutes())%60))
		} else {
			sb.WriteString("Fast duration target reached — eating window is open.\n")
		}
	} else {
		sb.WriteString("Currently fasting: NO (not in an active fast)\n")
	}

	sb.WriteString(fmt.Sprintf("Fasting streak: %d days\n", streak))

	// Include dietary preferences for meal suggestion context
	var dietType string
	var allergies []string
	h.pool.QueryRow(ctx,
		"SELECT diet_type, allergies FROM user_preferences WHERE user_id = $1", userID,
	).Scan(&dietType, &allergies)
	if dietType != "" && dietType != "omnivore" {
		sb.WriteString(fmt.Sprintf("Diet type: %s\n", dietType))
	}
	if len(allergies) > 0 {
		sb.WriteString(fmt.Sprintf("Allergies/intolerances: %s\n", strings.Join(allergies, ", ")))
	}

	return sb.String()
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req chatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, errors.New("content is required"))
		return
	}

	if len(req.Content) > 2000 {
		writeError(w, http.StatusBadRequest, errors.New("content must be 2000 characters or less"))
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	ctx := r.Context()

	if h.ai == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("AI coach is not configured"))
		return
	}

	plan := auth.GetPlan(ctx)
	if plan == "free" {
		var msgCount int
		_ = h.pool.QueryRow(ctx,
			`SELECT COUNT(*)::int FROM coach_messages WHERE user_id = $1 AND role = 'user' AND created_at::date = CURRENT_DATE`,
			userID).Scan(&msgCount)
		if msgCount > 5 {
			writeError(w, http.StatusTooManyRequests, errors.New("You've reached your daily coach limit. Upgrade to Premium for unlimited conversations."))
			return
		}
	}

	_, err = h.q.SaveCoachMessage(ctx, sqlc.SaveCoachMessageParams{
		UserID:  userID,
		Role:    "user",
		Content: req.Content,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save user message: %w", err))
		return
	}

	chatMessages, err := h.buildActiveContext(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("build context: %w", err))
		return
	}

	profile, err := h.q.GetProfile(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get profile: %w", err))
		return
	}

	goals, err := h.q.GetGoals(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get goals: %w", err))
		return
	}

	prefsCtx := h.fetchPreferencesContext(ctx, userID)
	stepsCtx := h.fetchStepsContext(ctx, userID)

	// Build user memory context
	var memoryCtx strings.Builder
	memories, memErr := LoadAllMemories(ctx, h.pool, userID)
	if memErr == nil && len(memories) > 0 {
		for _, m := range memories {
			memoryCtx.WriteString(fmt.Sprintf("- [%s] %s\n", m.Category, m.Content))
		}
	}

	// Load user coach notes
	var coachNotes string
	h.pool.QueryRow(ctx,
		"SELECT COALESCE(coach_notes, '') FROM user_profiles WHERE user_id = $1", userID,
	).Scan(&coachNotes)

	coachPrompt := admin.GetSettingDefault(h.pool, ctx, "prompt_coach", admin.DefaultPrompts["prompt_coach"])

	var identityAspiration string
	h.pool.QueryRow(ctx,
		`SELECT COALESCE(identity_aspiration, '') FROM user_profiles WHERE user_id = $1`, userID,
	).Scan(&identityAspiration)

	systemPrompt := fmt.Sprintf(
		coachPrompt,
		profile.Name,
		profileContext(profile, goals),
		prefsCtx,
		stepsCtx,
		memoryCtx.String(),
		coachNotes,
	)

	if identityAspiration != "" {
		systemPrompt += fmt.Sprintf("\n\nUser's Identity Aspiration: %s", identityAspiration)
	}

	tools := h.agentTools()

	// Agentic loop — max 3 iterations to prevent runaway calls
	const maxIterations = 3
	var finalResponse string

	for iteration := 0; iteration < maxIterations; iteration++ {
		agentResp, err := h.ai.ChatAgent(systemPrompt, chatMessages, tools)
		if err != nil {
			slog.Error("ChatAgent call failed", "iteration", iteration, "error", err, "msg_count", len(chatMessages), "tools_count", len(tools))
			writeError(w, http.StatusInternalServerError, fmt.Errorf("chat: %w", err))
			return
		}

		slog.Info("ChatAgent response", "iteration", iteration, "content_len", len(agentResp.Content), "tool_calls", len(agentResp.ToolCalls))

		// No tool calls — we have the final text response
		if len(agentResp.ToolCalls) == 0 {
			finalResponse = agentResp.Content
			break
		}

		toolNames := make([]string, len(agentResp.ToolCalls))
		for i, tc := range agentResp.ToolCalls {
			toolNames[i] = tc.Name
		}
		slog.Info("agent requesting tools", "iteration", iteration+1, "count", len(agentResp.ToolCalls))
		syslog.Info("ai", "Coach agent tool call", map[string]any{"user_id": userID, "user_name": profile.Name, "tools": toolNames, "iteration": iteration + 1})

		// Append assistant message with tool calls to the running context
		chatMessages = append(chatMessages, ai.ChatMessage{
			Role:      "assistant",
			Content:   agentResp.Content,
			ToolCalls: agentResp.ToolCalls,
		})

		// Execute each tool and append results
		for _, tc := range agentResp.ToolCalls {
			result := h.executeTool(ctx, userID, tc.Name, tc.Args)
			slog.Info("tool result", "tool", tc.Name, "result", result)
			chatMessages = append(chatMessages, ai.ChatMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}

		// On the last iteration, do a final call to get the text response
		if iteration == maxIterations-1 {
			finalResp, err := h.ai.ChatAgent(systemPrompt, chatMessages, tools)
			if err != nil {
				writeError(w, http.StatusInternalServerError, fmt.Errorf("final chat: %w", err))
				return
			}
			finalResponse = finalResp.Content
		}
	}

	if finalResponse == "" {
		finalResponse = "Done! I've taken care of that for you."
	}

	saved, err := h.q.SaveCoachMessage(ctx, sqlc.SaveCoachMessageParams{
		UserID:  userID,
		Role:    "assistant",
		Content: finalResponse,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save assistant message: %w", err))
		return
	}

	syslog.Info("ai", "Coach message sent", map[string]any{"user_id": userID, "user_name": profile.Name, "date": time.Now().Format("2006-01-02"), "response_len": len(finalResponse)})

	writeJSON(w, http.StatusCreated, apiResponse{Data: chatMessageResponse{
		ID:        saved.ID,
		Role:      saved.Role,
		Content:   saved.Content,
		CreatedAt: saved.CreatedAt.Format(time.RFC3339),
	}})
}

// fetchURL fetches a URL and returns its text content (HTML stripped).
// Limited to 3000 characters to keep AI context manageable.
func isBlockedHost(host string) bool {
	ips, err := net.LookupIP(host)
	if err != nil {
		return true
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return true
		}
	}
	return false
}

func (h *Handler) fetchURL(rawURL string) (string, error) {
	// Validate URL
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid URL")
	}

	if isBlockedHost(parsed.Hostname()) {
		return "", fmt.Errorf("URL resolves to a blocked address")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if isBlockedHost(req.URL.Hostname()) {
				return fmt.Errorf("redirect to blocked host")
			}
			return nil
		},
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Joules-Bot/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Read body (cap at 512 KB to avoid giant pages)
	limited := io.LimitReader(resp.Body, 512*1024)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	text := stripHTML(string(body))

	// Truncate to 3000 chars
	textRunes := []rune(text)
	if len(textRunes) > 3000 {
		text = string(textRunes[:3000]) + "\n[content truncated]"
	}
	return text, nil
}

var (
	reHTMLTags     = regexp.MustCompile(`<[^>]+>`)
	reWhitespace   = regexp.MustCompile(`\s{2,}`)
	reHTMLEntities = regexp.MustCompile(`&[a-zA-Z]+;|&#\d+;`)
)

func stripHTML(html string) string {
	// Remove script and style blocks entirely
	re := regexp.MustCompile(`(?si)<(script|style)[^>]*>.*?</(script|style)>`)
	text := re.ReplaceAllString(html, " ")
	text = reHTMLTags.ReplaceAllString(text, " ")
	text = reHTMLEntities.ReplaceAllString(text, " ")
	text = reWhitespace.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// lookupNutrition checks the DB cache for a food, then falls back to web search.
// Results are stored in nutrition_cache for future queries.
func (h *Handler) lookupNutrition(ctx context.Context, foodName string) (string, error) {
	// 1. Check cache
	cached, err := h.q.GetNutritionCache(ctx, foodName)
	if err == nil {
		slog.Info("nutrition cache hit", "food", foodName)
		return fmt.Sprintf(
			`Nutrition info for "%s" (cached, source: %s):
Serving: %s
Calories: %d kcal | Protein: %.1fg | Carbs: %.1fg | Fat: %.1fg | Fiber: %.1fg`,
			cached.Name, cached.Source, cached.ServingSize,
			cached.Calories,
			numericToFloat(cached.ProteinG),
			numericToFloat(cached.CarbsG),
			numericToFloat(cached.FatG),
			numericToFloat(cached.FiberG),
		), nil
	}

	// 2. Web search fallback (Tavily)
	if h.cfg == nil || h.cfg.TavilyAPIKey == "" {
		return fmt.Sprintf("No cached data found for \"%s\" and web search is not configured (TAVILY_API_KEY not set).", foodName), nil
	}

	slog.Info("nutrition cache miss, searching web", "food", foodName)
	searchResult, err := ai.SearchWeb(h.cfg.TavilyAPIKey, foodName+" calories nutrition facts per serving")
	if err != nil {
		return "", fmt.Errorf("web search failed: %w", err)
	}

	// 3. Ask AI to parse the search result into structured nutrition data
	nutritionPrompt := admin.GetSettingDefault(h.pool, ctx, "prompt_nutrition_lookup", admin.DefaultPrompts["prompt_nutrition_lookup"])
	parsePrompt := fmt.Sprintf(nutritionPrompt, foodName, searchResult)

	parsed, err := h.ai.Chat(parsePrompt, nil)
	if err != nil {
		return searchResult, nil // return raw search if AI parse fails
	}

	parsed = strings.TrimSpace(parsed)
	if strings.HasPrefix(parsed, "```") {
		if idx := strings.Index(parsed, "\n"); idx != -1 {
			parsed = parsed[idx+1:]
		}
		parsed = strings.TrimSuffix(parsed, "```")
		parsed = strings.TrimSpace(parsed)
	}

	var nutrition struct {
		Name        string  `json:"name"`
		Calories    int     `json:"calories"`
		ProteinG    float64 `json:"protein_g"`
		CarbsG      float64 `json:"carbs_g"`
		FatG        float64 `json:"fat_g"`
		FiberG      float64 `json:"fiber_g"`
		ServingSize string  `json:"serving_size"`
		Error       string  `json:"error"`
	}

	if err := json.Unmarshal([]byte(parsed), &nutrition); err != nil || nutrition.Error != "" {
		return fmt.Sprintf("Found web data for \"%s\" but couldn't parse it precisely:\n%s", foodName, searchResult), nil
	}

	if nutrition.Name == "" {
		nutrition.Name = foodName
	}
	if nutrition.ServingSize == "" {
		nutrition.ServingSize = "1 serving"
	}

	// 4. Store in cache for future use
	proteinN := pgtype.Numeric{}
	_ = proteinN.Scan(fmt.Sprintf("%.2f", nutrition.ProteinG))
	carbsN := pgtype.Numeric{}
	_ = carbsN.Scan(fmt.Sprintf("%.2f", nutrition.CarbsG))
	fatN := pgtype.Numeric{}
	_ = fatN.Scan(fmt.Sprintf("%.2f", nutrition.FatG))
	fiberN := pgtype.Numeric{}
	_ = fiberN.Scan(fmt.Sprintf("%.2f", nutrition.FiberG))

	_, cacheErr := h.q.UpsertNutritionCache(ctx, sqlc.UpsertNutritionCacheParams{
		Query:       foodName,
		Name:        nutrition.Name,
		Calories:    int32(nutrition.Calories),
		ProteinG:    proteinN,
		CarbsG:      carbsN,
		FatG:        fatN,
		FiberG:      fiberN,
		ServingSize: nutrition.ServingSize,
		Source:      "web",
	})
	if cacheErr != nil {
		slog.Warn("failed to cache nutrition data", "food", foodName, "error", cacheErr)
	} else {
		slog.Info("nutrition data cached", "food", foodName)
	}

	return fmt.Sprintf(
		`Nutrition info for "%s" (source: web, now cached):
Serving: %s
Calories: %d kcal | Protein: %.1fg | Carbs: %.1fg | Fat: %.1fg | Fiber: %.1fg`,
		nutrition.Name, nutrition.ServingSize,
		nutrition.Calories, nutrition.ProteinG, nutrition.CarbsG, nutrition.FatG, nutrition.FiberG,
	), nil
}

func (h *Handler) GetRemindersAPI(w http.ResponseWriter, r *http.Request) {
	userID, err := getCoachUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	reminders, err := GetReminders(r.Context(), h.pool, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get reminders: %w", err))
		return
	}
	if reminders == nil {
		reminders = []ReminderEntry{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: reminders})
}

func (h *Handler) ToggleReminderAPI(w http.ResponseWriter, r *http.Request) {
	userID, err := getCoachUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	reminderID := chi.URLParam(r, "id")
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := ToggleReminder(r.Context(), h.pool, userID, reminderID, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("toggle reminder: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "updated"}})
}

func (h *Handler) DeleteReminderAPI(w http.ResponseWriter, r *http.Request) {
	userID, err := getCoachUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	reminderID := chi.URLParam(r, "id")
	if err := DeleteReminder(r.Context(), h.pool, userID, reminderID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete reminder: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "deleted"}})
}
