package coach

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"joule/internal/ai"
	"joule/internal/db/sqlc"
)

type contextKey string

type Handler struct {
	q  *sqlc.Queries
	ai ai.Client
}

func NewHandler(q *sqlc.Queries, aiClient ai.Client) *Handler {
	return &Handler{q: q, ai: aiClient}
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
	return r.Context().Value(contextKey("userID")).(string)
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
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
		"User profile: Name: %s, Age: %d, Sex: %s, Weight: %.1fkg, Height: %.1fcm, Activity: %s\nGoals: Objective: %s, Diet plan: %s, Daily calories: %d, Protein: %dg, Carbs: %dg, Fat: %dg",
		profile.Name,
		age,
		sex,
		numericToFloat(profile.WeightKg),
		numericToFloat(profile.HeightCm),
		activity,
		goals.Objective,
		goals.DietPlan,
		goals.DailyCalorieTarget,
		goals.DailyProteinG,
		goals.DailyCarbsG,
		goals.DailyFatG,
	)
}

func (h *Handler) GetTips(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	today := time.Now().Format("2006-01-02")
	cacheKey := userID + ":" + today

	if cached, ok := tipsCache.Load(cacheKey); ok {
		writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"tips": cached.(string)}})
		return
	}

	ctx := r.Context()

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

	prompt := fmt.Sprintf(
		`You are a friendly health and nutrition coach. Based on the user's profile and today's data, provide 3-4 brief, actionable daily tips. Keep each tip to one sentence. Be encouraging and specific. Use markdown formatting.

%s
Today's intake: %d/%d calories, %.0f/%dg protein, %dml water`,
		profileContext(profile, goals),
		summary.TotalCalories,
		goals.DailyCalorieTarget,
		summary.TotalProtein,
		goals.DailyProteinG,
		summary.TotalWaterMl,
	)

	tips, err := h.ai.Chat(prompt, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("generate tips: %w", err))
		return
	}

	tipsCache.Store(cacheKey, tips)

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

	systemPrompt := fmt.Sprintf(
		`You are Joule's AI health coach. You help users with nutrition, exercise, and healthy lifestyle advice. Be friendly, concise, and actionable. Use markdown formatting when helpful.

%s`,
		profileContext(profile, goals),
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

	response, err := h.ai.Chat(systemPrompt, chatMessages)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("chat: %w", err))
		return
	}

	saved, err := h.q.SaveCoachMessage(ctx, sqlc.SaveCoachMessageParams{
		UserID:  userID,
		Role:    "assistant",
		Content: response,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save assistant message: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: chatMessageResponse{
		ID:        saved.ID,
		Role:      saved.Role,
		Content:   saved.Content,
		CreatedAt: saved.CreatedAt.Format(time.RFC3339),
	}})
}
