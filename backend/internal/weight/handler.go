package weight

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

func getUserID(r *http.Request) string {
	return r.Context().Value(contextKey("userID")).(string)
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

func (h *Handler) LogWeight(w http.ResponseWriter, r *http.Request) {
	var req LogWeightRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
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
		date = time.Now()
	}

	userID := getUserID(r)

	logged, err := h.q.LogWeight(r.Context(), sqlc.LogWeightParams{
		UserID:   userID,
		Date:     date,
		WeightKg: floatToNumeric(req.WeightKg),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("log weight: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: WeightResponse{
		Date:     logged.Date.Format("2006-01-02"),
		WeightKg: numericToFloat(logged.WeightKg),
	}})
}

func (h *Handler) GetWeightHistory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from := time.Now().AddDate(0, 0, -30)
	to := time.Now()

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
