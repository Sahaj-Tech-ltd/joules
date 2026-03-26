package exercise

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"joule/internal/db/sqlc"
)

type contextKey string

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
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

func getUserID(r *http.Request) string {
	return r.Context().Value(contextKey("userID")).(string)
}

func (h *Handler) LogExercise(w http.ResponseWriter, r *http.Request) {
	var req LogExerciseRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
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
		timestamp = time.Now()
	}

	weightKg := req.WeightKg
	if weightKg == 0 {
		weightKg = 70.0
	}

	met := findMET(req.Name)
	calories := calculateCalories(met, weightKg, req.DurationMin)

	userID := getUserID(r)

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
	userID := getUserID(r)

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
		date = time.Now()
	}

	exercises, err := h.q.GetExercisesByDate(r.Context(), sqlc.GetExercisesByDateParams{
		UserID:    userID,
		Timestamp: date,
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
