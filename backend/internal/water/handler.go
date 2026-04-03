package water

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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

type LogWaterRequest struct {
	AmountMl int32  `json:"amount_ml"`
	Date     string `json:"date"`
}

type WaterLogResponse struct {
	Date     string `json:"date"`
	AmountMl int32  `json:"amount_ml"`
}

type WaterTotalResponse struct {
	Date    string `json:"date"`
	TotalMl int32  `json:"total_ml"`
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

func getUserID(r *http.Request) string {
	return r.Context().Value(auth.ContextUserID).(string)
}

func tzNow(tz string) time.Time {
	if loc, err := time.LoadLocation(tz); err == nil {
		return time.Now().In(loc)
	}
	return time.Now()
}

func (h *Handler) LogWater(w http.ResponseWriter, r *http.Request) {
	var req LogWaterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.AmountMl <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("amount_ml must be greater than 0"))
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

	userID := getUserID(r)

	logged, err := h.q.LogWater(r.Context(), sqlc.LogWaterParams{
		UserID:   userID,
		Date:     date,
		AmountMl: req.AmountMl,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("log water: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: WaterLogResponse{
		Date:     logged.Date.Format("2006-01-02"),
		AmountMl: logged.AmountMl,
	}})
}

func (h *Handler) GetWaterByDate(w http.ResponseWriter, r *http.Request) {
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
		date = tzNow(r.Header.Get("X-Timezone"))
	}

	userID := getUserID(r)

	total, err := h.q.GetWaterByDate(r.Context(), sqlc.GetWaterByDateParams{
		UserID: userID,
		Date:   date,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get water by date: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: WaterTotalResponse{
		Date:    date.Format("2006-01-02"),
		TotalMl: total,
	}})
}
