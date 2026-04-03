package exercise

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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

type LogExerciseRequest struct {
	Name        string  `json:"name"`
	DurationMin int32   `json:"duration_min"`
	WeightKg    float64 `json:"weight_kg"`
	Timestamp   string  `json:"timestamp"`
}

type ExerciseResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DurationMin    int32  `json:"duration_min"`
	CaloriesBurned int32  `json:"calories_burned"`
	Timestamp      string `json:"timestamp"`
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

func tzNow(tz string) time.Time {
	if loc, err := time.LoadLocation(tz); err == nil {
		return time.Now().In(loc)
	}
	return time.Now()
}

func (h *Handler) LogExercise(w http.ResponseWriter, r *http.Request) {
	var req LogExerciseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}

	if req.DurationMin <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("duration_min must be greater than 0"))
		return
	}

	var timestamp time.Time
	if req.Timestamp != "" {
		var err error
		timestamp, err = time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid timestamp format: %w", err))
			return
		}
	} else {
		timestamp = tzNow(r.Header.Get("X-Timezone"))
	}

	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	weightKg := req.WeightKg
	if weightKg == 0 {
		var profileWeight float64
		err := h.pool.QueryRow(r.Context(),
			"SELECT COALESCE(weight_kg, 0) FROM user_profiles WHERE user_id = $1", userID,
		).Scan(&profileWeight)
		if err != nil || profileWeight == 0 {
			weightKg = 70.0
		} else {
			weightKg = profileWeight
		}
	}

	met := FindMET(req.Name)
	calories := CalculateCalories(met, weightKg, req.DurationMin)

	logged, err := h.q.LogExercise(r.Context(), sqlc.LogExerciseParams{
		UserID:         userID,
		Timestamp:      timestamp,
		Name:           req.Name,
		DurationMin:    req.DurationMin,
		CaloriesBurned: calories,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("log exercise: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: ExerciseResponse{
		ID:             logged.ID,
		Name:           logged.Name,
		DurationMin:    logged.DurationMin,
		CaloriesBurned: logged.CaloriesBurned,
		Timestamp:      logged.Timestamp.Format(time.RFC3339),
	}})
}

func (h *Handler) GetExercisesByDate(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	tz := r.Header.Get("X-Timezone")
	if tz == "" {
		tz = "UTC"
	}

	dateStr := r.URL.Query().Get("date")

	var date time.Time
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date format: %w", err))
			return
		}
	} else {
		date = tzNow(tz)
	}

	exercises, err := h.q.GetExercisesByDate(r.Context(), sqlc.GetExercisesByDateParams{
		UserID:    userID,
		Timestamp: date,
		Column3:   tz,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get exercises: %w", err))
		return
	}

	resp := make([]ExerciseResponse, 0, len(exercises))
	for _, e := range exercises {
		resp = append(resp, ExerciseResponse{
			ID:             e.ID,
			Name:           e.Name,
			DurationMin:    e.DurationMin,
			CaloriesBurned: e.CaloriesBurned,
			Timestamp:      e.Timestamp.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}
