package weight

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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

type LogWeightRequest struct {
	WeightKg float64 `json:"weight_kg"`
	Date     string  `json:"date"`
}

type WeightResponse struct {
	Date     string  `json:"date"`
	WeightKg float64 `json:"weight_kg"`
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

func floatToNumeric(f float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func tzNow(tz string) time.Time {
	if loc, err := time.LoadLocation(tz); err == nil {
		return time.Now().In(loc)
	}
	return time.Now()
}

func (h *Handler) LogWeight(w http.ResponseWriter, r *http.Request) {
	var req LogWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.WeightKg <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("weight_kg must be greater than 0"))
		return
	}

	var date time.Time
	if req.Date != "" {
		var err error
		date, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date format: %w", err))
			return
		}
	} else {
		date = tzNow(r.Header.Get("X-Timezone"))
	}

	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	logged, err := h.q.LogWeight(r.Context(), sqlc.LogWeightParams{
		UserID:   userID,
		Date:     date,
		WeightKg: floatToNumeric(req.WeightKg),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("log weight: %w", err))
		return
	}

	_, _ = h.pool.Exec(r.Context(),
		"UPDATE user_profiles SET weight_kg = $1, updated_at = NOW() WHERE user_id = $2",
		req.WeightKg, userID,
	)

	writeJSON(w, http.StatusCreated, apiResponse{Data: WeightResponse{
		Date:     logged.Date.Format("2006-01-02"),
		WeightKg: numericToFloat(logged.WeightKg),
	}})
}

func (h *Handler) GetWeightHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	now := tzNow(r.Header.Get("X-Timezone"))
	from := now.AddDate(0, 0, -30)
	to := now

	if fromStr != "" {
		var err error
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid from date format: %w", err))
			return
		}
	}

	if toStr != "" {
		var err error
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid to date format: %w", err))
			return
		}
	}

	weights, err := h.q.GetWeightHistory(r.Context(), sqlc.GetWeightHistoryParams{
		UserID: userID,
		Date:   from,
		Date_2: to,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get weight history: %w", err))
		return
	}

	var resp []WeightResponse
	for _, w := range weights {
		resp = append(resp, WeightResponse{
			Date:     w.Date.Format("2006-01-02"),
			WeightKg: numericToFloat(w.WeightKg),
		})
	}

	if resp == nil {
		resp = []WeightResponse{}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}
