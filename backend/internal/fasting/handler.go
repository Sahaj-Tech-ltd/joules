package fasting

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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

func windowToHours(window string) int {
	switch window {
	case "16:8":
		return 8
	case "18:6":
		return 6
	case "20:4":
		return 4
	case "omad":
		return 1
	default:
		return 8
	}
}

func pgTimeToHHMM(t pgtype.Time) string {
	if !t.Valid {
		return "12:00"
	}
	h := t.Microseconds / 3600_000_000
	m := (t.Microseconds % 3600_000_000) / 60_000_000
	return fmt.Sprintf("%02d:%02d", h, m)
}

type FastingStatusResponse struct {
	IsFasting         bool    `json:"is_fasting"`
	FastStartTime     *string `json:"fast_start_time"`
	EatingWindowStart string  `json:"eating_window_start"`
	EatingWindowHours int     `json:"eating_window_hours"`
	FastingHours      int     `json:"fasting_hours"`
	SecondsElapsed    int64   `json:"seconds_elapsed"`
	SecondsRemaining  int64   `json:"seconds_remaining"`
	FastingStreak     int     `json:"fasting_streak"`
	FastingWindow     string  `json:"fasting_window"`
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	ctx := r.Context()

	goals, err := h.q.GetGoals(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get goals: %w", err))
		return
	}

	eatingStart := pgTimeToHHMM(goals.EatingWindowStart)

	fastingWindow := "16:8"
	if goals.FastingWindow != nil {
		fastingWindow = *goals.FastingWindow
	}

	eatingHours := windowToHours(fastingWindow)
	fastingHours := 24 - eatingHours
	streak := 0
	if goals.FastingStreak != nil {
		streak = int(*goals.FastingStreak)
	}

	resp := FastingStatusResponse{
		IsFasting:         false,
		EatingWindowStart: eatingStart,
		EatingWindowHours: eatingHours,
		FastingHours:      fastingHours,
		FastingStreak:     streak,
		FastingWindow:     fastingWindow,
	}

	if goals.CurrentFastStart.Valid {
		resp.IsFasting = true
		fastStart := goals.CurrentFastStart.Time.Format(time.RFC3339)
		resp.FastStartTime = &fastStart
		elapsed := time.Since(goals.CurrentFastStart.Time).Seconds()
		resp.SecondsElapsed = int64(elapsed)
		totalFastingSecs := float64(fastingHours * 3600)
		remaining := totalFastingSecs - elapsed
		if remaining < 0 {
			remaining = 0
		}
		resp.SecondsRemaining = int64(remaining)
	} else {
		resp.SecondsElapsed = 0
		resp.SecondsRemaining = 0
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *Handler) StartFast(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if err := h.q.StartFast(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("start fast: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "fast started"}})
}

func (h *Handler) BreakFast(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	ctx := r.Context()

	goals, err := h.q.GetGoals(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get goals: %w", err))
		return
	}

	streak := int32(0)
	if goals.FastingStreak != nil {
		streak = *goals.FastingStreak
	}

	fastingWindow := "16:8"
	if goals.FastingWindow != nil {
		fastingWindow = *goals.FastingWindow
	}
	fastingHours := float64(24 - windowToHours(fastingWindow))
	requiredSecs := fastingHours * 3600

	if goals.CurrentFastStart.Valid {
		elapsed := time.Since(goals.CurrentFastStart.Time).Seconds()
		if elapsed >= requiredSecs {
			streak++
		}
	}

	if err := h.q.BreakFast(ctx, sqlc.BreakFastParams{
		UserID:        userID,
		FastingStreak: &streak,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("break fast: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"message":        "fast broken",
		"fasting_streak": streak,
	}})
}

type UpdateWindowRequest struct {
	EatingWindowStart string `json:"eating_window_start"` // "HH:MM"
}

func (h *Handler) UpdateWindow(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var req UpdateWindowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.EatingWindowStart == "" {
		req.EatingWindowStart = "12:00"
	}

	parts := strings.Split(req.EatingWindowStart, ":")
	if len(parts) != 2 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid time format, expected HH:MM"))
		return
	}
	var hh, mm int
	fmt.Sscanf(req.EatingWindowStart, "%d:%d", &hh, &mm)
	pgTime := pgtype.Time{
		Microseconds: int64(hh)*3600*1000000 + int64(mm)*60*1000000,
		Valid:        true,
	}

	if err := h.q.UpdateEatingWindow(r.Context(), sqlc.UpdateEatingWindowParams{
		UserID:            userID,
		EatingWindowStart: pgTime,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("update window: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{
		"message": "eating window updated",
	}})
}
