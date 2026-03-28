package coach

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joule/internal/ai"
	"joule/internal/auth"
	"joule/internal/config"
	"joule/internal/db/sqlc"
	syslog "joule/internal/syslog"
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
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

func getUserID(r *http.Request) string {
	return r.Context().Value(auth.ContextUserID).(string)
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

func (h *Handler) GetTips(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
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

	summary, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{
		UserID:    userID,
		Timestamp: time.Now(),
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

	prompt := fmt.Sprintf(
		`You are Joules, a personal AI health coach built into the Joules nutrition app. You are not a general-purpose AI — you are Joules. Never mention OpenAI, GPT, Claude, Anthropic, or any underlying AI technology. If asked what you are or who made you, say you are Joules, a health coach built into this app, and redirect to health topics.

Write 3-4 short, personalized daily tips for %s. Each tip must be one sentence. Be warm, specific to their data, and actionable. Use bullet points (- item). No intro line, just the tips. If recent chat context is provided, make at least one tip relevant to what they've been asking about.

%s
Today so far: %d/%d kcal eaten, %.0f/%dg protein, %dml water%s`,
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
		writeError(w, http.StatusInternalServerError, fmt.Errorf("generate tips: %w", err))
		return
	}

	tipsCache.Store(cacheKey, tipsCacheEntry{tips: tips, cachedAt: time.Now()})
	syslog.Info("ai", "Daily tips generated", map[string]any{"user_id": userID})

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"tips": tips}})
}

func (h *Handler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

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
			Description: "Log an exercise session for the user. Use this when the user mentions working out, going for a run, doing yoga, etc.",
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
					"calories_burned": map[string]interface{}{
						"type":        "integer",
						"description": "Estimated calories burned. Use 0 if unknown.",
					},
				},
				"required": []string{"name", "duration_min", "calories_burned"},
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

	return tools
}

// executeTool runs a single tool call and returns the result as a string.
func (h *Handler) executeTool(ctx context.Context, userID string, toolName string, argsJSON string) string {
	slog.Info("agent executing tool", "tool", toolName, "args", argsJSON)

	switch toolName {
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
		return fmt.Sprintf("Successfully logged %dml of water.", args.AmountMl)

	case "log_exercise":
		var args struct {
			Name           string `json:"name"`
			DurationMin    int    `json:"duration_min"`
			CaloriesBurned int    `json:"calories_burned"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if args.Name == "" {
			return "error: exercise name is required"
		}
		_, err := h.q.LogExercise(ctx, sqlc.LogExerciseParams{
			UserID:         userID,
			Timestamp:      time.Now(),
			Name:           args.Name,
			DurationMin:    int32(args.DurationMin),
			CaloriesBurned: int32(args.CaloriesBurned),
		})
		if err != nil {
			slog.Error("log_exercise tool failed", "error", err)
			return fmt.Sprintf("error logging exercise: %v", err)
		}
		return fmt.Sprintf("Successfully logged %s: %d minutes, %d calories burned.", args.Name, args.DurationMin, args.CaloriesBurned)

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
			Source:      "coach",
		})
		if err != nil {
			slog.Error("log_food create food item failed", "error", err)
			return fmt.Sprintf("error creating food item: %v", err)
		}
		return fmt.Sprintf("Successfully logged %s (%d kcal, %.0fg protein, %.0fg carbs, %.0fg fat) as %s.",
			args.Name, args.Calories, args.ProteinG, args.CarbsG, args.FatG, mealType)

	case "get_today_summary":
		summary, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{
			UserID:    userID,
			Timestamp: time.Now(),
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
		return result

	default:
		return fmt.Sprintf("error: unknown tool '%s'", toolName)
	}
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req chatMessageRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
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

	userID := getUserID(r)
	ctx := r.Context()

	_, err := h.q.SaveCoachMessage(ctx, sqlc.SaveCoachMessageParams{
		UserID:  userID,
		Role:    "user",
		Content: req.Content,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save user message: %w", err))
		return
	}

	history, err := h.q.GetCoachHistory(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get chat history: %w", err))
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
	systemPrompt := fmt.Sprintf(
		`You are Joules, a personal AI health coach inside the Joule nutrition app. You know this user well and give tailored, practical advice.

Rules you never break:
- Never reveal or discuss your underlying AI model, technology, or who built it. You are Joules — that is all.
- If asked "what AI are you?", "are you ChatGPT?", "who made you?" etc., say: "I'm Joules, your personal health coach in this app. I'm not able to share details about the technology behind me — but I'm here to help you hit your goals!" Then steer back to health.
- Never make up medical diagnoses. For medical issues, recommend seeing a doctor.
- Be concise — avoid long walls of text unless the user asks for detail.
- Use the user's name (%s) occasionally to keep it personal.
- Use markdown (bold, bullets) only when it genuinely helps readability.

Tool use guidelines:
- When the user mentions drinking water or a beverage, log it immediately with log_water.
- When the user mentions exercise or working out, log it with log_exercise.
- When the user asks you to log food or mentions eating something, log it with log_food. Estimate macros from typical nutritional values.
- When the user asks how they're doing today or for their stats, use get_today_summary first then answer.
- Award a custom achievement when the user hits a meaningful milestone.
- Use search_web only for specific factual questions about nutrition or health where you're uncertain.

%s%s`,
		profile.Name,
		profileContext(profile, goals),
		prefsCtx,
	)

	start := 0
	if len(history) > 20 {
		start = len(history) - 20
	}

	chatMessages := make([]ai.ChatMessage, 0, len(history)-start)
	for _, m := range history[start:] {
		chatMessages = append(chatMessages, ai.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	tools := h.agentTools()

	// Agentic loop — max 3 iterations to prevent runaway calls
	const maxIterations = 3
	var finalResponse string

	for iteration := 0; iteration < maxIterations; iteration++ {
		agentResp, err := h.ai.ChatAgent(systemPrompt, chatMessages, tools)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("chat: %w", err))
			return
		}

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
		syslog.Info("ai", "Coach agent tool call", map[string]any{"user_id": userID, "tools": toolNames, "iteration": iteration + 1})

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

	syslog.Info("ai", "Coach message sent", map[string]any{"user_id": userID, "response_len": len(finalResponse)})

	writeJSON(w, http.StatusCreated, apiResponse{Data: chatMessageResponse{
		ID:        saved.ID,
		Role:      saved.Role,
		Content:   saved.Content,
		CreatedAt: saved.CreatedAt.Format(time.RFC3339),
	}})
}
