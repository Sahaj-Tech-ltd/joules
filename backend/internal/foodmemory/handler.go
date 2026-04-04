package foodmemory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
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
	slog.Error("foodmemory error", "status", status, "error", err)
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

type foodMemoryEntry struct {
	ID              string   `json:"id"`
	FoodName        string   `json:"food_name"`
	CanonicalName   *string  `json:"canonical_name"`
	Calories        float32  `json:"calories"`
	Protein         float32  `json:"protein"`
	Carbs           float32  `json:"carbs"`
	Fat             float32  `json:"fat"`
	Fiber           float32  `json:"fiber"`
	ServingSize     *float32 `json:"serving_size"`
	ServingUnit     *string  `json:"serving_unit"`
	CorrectionCount int      `json:"correction_count"`
	Source          string   `json:"source"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, food_name, canonical_name, calories, protein, carbs, fat, fiber,
		        serving_size, serving_unit, correction_count, source, created_at, updated_at
		 FROM user_food_memory WHERE user_id = $1 ORDER BY correction_count DESC, updated_at DESC`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var results []foodMemoryEntry
	for rows.Next() {
		var e foodMemoryEntry
		var createdAt, updatedAt time.Time
		err := rows.Scan(&e.ID, &e.FoodName, &e.CanonicalName, &e.Calories, &e.Protein,
			&e.Carbs, &e.Fat, &e.Fiber, &e.ServingSize, &e.ServingUnit,
			&e.CorrectionCount, &e.Source, &createdAt, &updatedAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		e.CreatedAt = createdAt.Format(time.RFC3339)
		e.UpdatedAt = updatedAt.Format(time.RFC3339)
		results = append(results, e)
	}
	if results == nil {
		results = []foodMemoryEntry{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: results})
}

type FoodMemoryMatch struct {
	CanonicalName string  `json:"canonical_name"`
	Calories      float32 `json:"calories"`
	Protein       float32 `json:"protein"`
	Carbs         float32 `json:"carbs"`
	Fat           float32 `json:"fat"`
	Fiber         float32 `json:"fiber"`
}

func GetFoodMemory(pool *pgxpool.Pool, userID, foodName string) (*FoodMemoryMatch, error) {
	var match FoodMemoryMatch
	err := pool.QueryRow(context.Background(),
		`SELECT COALESCE(canonical_name, food_name), calories, protein, carbs, fat, fiber
		 FROM user_food_memory
		 WHERE user_id = $1 AND (food_name ILIKE $2 OR canonical_name ILIKE $2)
		 ORDER BY correction_count DESC LIMIT 1`,
		userID, "%"+foodName+"%").Scan(&match.CanonicalName, &match.Calories, &match.Protein,
		&match.Carbs, &match.Fat, &match.Fiber)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &match, nil
}
