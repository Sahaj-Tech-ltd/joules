package identity

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/ai"
	"joules/internal/auth"
	"joules/internal/db/sqlc"
)

type Handler struct {
	q        *sqlc.Queries
	pool     *pgxpool.Pool
	aiClient ai.Client
}

func NewHandler(q *sqlc.Queries, pool *pgxpool.Pool, aiClient ai.Client) *Handler {
	return &Handler{q: q, pool: pool, aiClient: aiClient}
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
	slog.Error("identity error", "status", status, "error", err)
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

type quoteResponse struct {
	Quote string `json:"quote"`
	Date  string `json:"date"`
}

func (h *Handler) GetQuote(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	todayStr := today.Format("2006-01-02")

	var existingQuote string
	err = h.pool.QueryRow(r.Context(),
		`SELECT quote FROM identity_quotes WHERE user_id = $1 AND date = $2`,
		userID, todayStr).Scan(&existingQuote)
	if err == nil && existingQuote != "" {
		writeJSON(w, http.StatusOK, apiResponse{Data: quoteResponse{
			Quote: existingQuote,
			Date:  todayStr,
		}})
		return
	}

	var aspiration string
	_ = h.pool.QueryRow(r.Context(),
		`SELECT COALESCE(identity_aspiration, '') FROM user_profiles WHERE user_id = $1`,
		userID).Scan(&aspiration)

	var streakDays int
	_ = h.pool.QueryRow(r.Context(),
		`SELECT COALESCE(streak_days, 0) FROM user_stats WHERE user_id = $1`,
		userID).Scan(&streakDays)

	var recentFoods string
	rows, err := h.pool.Query(r.Context(),
		`SELECT DISTINCT m.name FROM meals m WHERE m.user_id = $1 AND m.timestamp >= NOW() - INTERVAL '3 days' ORDER BY m.timestamp DESC LIMIT 5`,
		userID)
	if err == nil {
		var foods []string
		for rows.Next() {
			var f string
			if rows.Scan(&f) == nil {
				foods = append(foods, f)
			}
		}
		rows.Close()
		if len(foods) > 0 {
			recentFoods = "Recent foods: " + foods[0]
			for i := 1; i < len(foods); i++ {
				recentFoods += ", " + foods[i]
			}
		}
	}

	systemPrompt := `You are a motivational coach for a nutrition tracking app called Joules. Generate a single personalized identity-based quote (1-2 sentences) that connects the user's health identity to their daily food choices. Be specific, warm, and empowering. Do not use clichés. Return ONLY the quote text, nothing else.`

	userMsg := fmt.Sprintf("User's identity aspiration: %q. Current streak: %d days. %s. Generate a fresh identity-based motivational quote for today.",
		aspiration, streakDays, recentFoods)

	quote, err := h.aiClient.Chat(systemPrompt, []ai.ChatMessage{
		{Role: "user", Content: userMsg},
	})
	if err != nil {
		slog.Error("identity quote generation failed", "error", err)
		quote = "Every healthy choice you make today is a vote for the person you want to become."
	}

	_, err = h.pool.Exec(r.Context(),
		`INSERT INTO identity_quotes (user_id, quote, date) VALUES ($1, $2, $3) ON CONFLICT (user_id, date) DO UPDATE SET quote = $2`,
		userID, quote, todayStr)
	if err != nil {
		slog.Error("failed to store identity quote", "error", err)
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: quoteResponse{
		Quote: quote,
		Date:  todayStr,
	}})
}
