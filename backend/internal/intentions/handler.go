package intentions

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	slog.Error("intentions error", "status", status, "error", err)
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

type intentionRow struct {
	ID               string  `json:"id"`
	MealType         string  `json:"meal_type"`
	TriggerText      string  `json:"trigger_text"`
	ActionText       string  `json:"action_text"`
	NotificationTime *string `json:"notification_time"`
	Enabled          bool    `json:"enabled"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func scanIntention(scanner interface {
	Scan(dest ...any) error
}) (intentionRow, error) {
	var r intentionRow
	var createdAt, updatedAt time.Time
	var notifTime *string
	err := scanner.Scan(&r.ID, &r.MealType, &r.TriggerText, &r.ActionText, &notifTime, &r.Enabled, &createdAt, &updatedAt)
	if err != nil {
		return r, err
	}
	r.NotificationTime = notifTime
	r.CreatedAt = createdAt.Format(time.RFC3339)
	r.UpdatedAt = updatedAt.Format(time.RFC3339)
	return r, nil
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, meal_type, trigger_text, action_text, notification_time, enabled, created_at, updated_at
		 FROM implementation_intentions WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var results []intentionRow
	for rows.Next() {
		item, err := scanIntention(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		results = append(results, item)
	}
	if results == nil {
		results = []intentionRow{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: results})
}

type createIntentionReq struct {
	MealType         string  `json:"meal_type"`
	TriggerText      string  `json:"trigger_text"`
	ActionText       string  `json:"action_text"`
	NotificationTime *string `json:"notification_time"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	var req createIntentionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.MealType == "" || req.TriggerText == "" || req.ActionText == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("meal_type, trigger_text, and action_text are required"))
		return
	}

	id := uuid.New().String()
	var item intentionRow
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO implementation_intentions (id, user_id, meal_type, trigger_text, action_text, notification_time)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, meal_type, trigger_text, action_text, notification_time, enabled, created_at, updated_at`,
		id, userID, req.MealType, req.TriggerText, req.ActionText, req.NotificationTime).Scan(
		&item.ID, &item.MealType, &item.TriggerText, &item.ActionText, &item.NotificationTime, &item.Enabled,
		new(time.Time), new(time.Time))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	item.CreatedAt = time.Now().Format(time.RFC3339)
	item.UpdatedAt = item.CreatedAt
	writeJSON(w, http.StatusCreated, apiResponse{Data: item})
}

type updateIntentionReq struct {
	MealType         *string `json:"meal_type"`
	TriggerText      *string `json:"trigger_text"`
	ActionText       *string `json:"action_text"`
	NotificationTime *string `json:"notification_time"`
	Enabled          *bool   `json:"enabled"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing intention id"))
		return
	}

	var req updateIntentionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var existing intentionRow
	err = h.pool.QueryRow(r.Context(),
		`SELECT id, meal_type, trigger_text, action_text, notification_time, enabled, created_at, updated_at
		 FROM implementation_intentions WHERE id = $1 AND user_id = $2`, id, userID).Scan(
		&existing.ID, &existing.MealType, &existing.TriggerText, &existing.ActionText,
		&existing.NotificationTime, &existing.Enabled, new(time.Time), new(time.Time))
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("intention not found"))
		return
	}

	mealType := existing.MealType
	triggerText := existing.TriggerText
	actionText := existing.ActionText
	notifTime := existing.NotificationTime
	enabled := existing.Enabled

	if req.MealType != nil {
		mealType = *req.MealType
	}
	if req.TriggerText != nil {
		triggerText = *req.TriggerText
	}
	if req.ActionText != nil {
		actionText = *req.ActionText
	}
	if req.NotificationTime != nil {
		notifTime = req.NotificationTime
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	var item intentionRow
	err = h.pool.QueryRow(r.Context(),
		`UPDATE implementation_intentions SET meal_type = $3, trigger_text = $4, action_text = $5, notification_time = $6, enabled = $7, updated_at = NOW()
		 WHERE id = $1 AND user_id = $2
		 RETURNING id, meal_type, trigger_text, action_text, notification_time, enabled, created_at, updated_at`,
		id, userID, mealType, triggerText, actionText, notifTime, enabled).Scan(
		&item.ID, &item.MealType, &item.TriggerText, &item.ActionText,
		&item.NotificationTime, &item.Enabled, new(time.Time), new(time.Time))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	item.CreatedAt = time.Now().Format(time.RFC3339)
	item.UpdatedAt = item.CreatedAt
	writeJSON(w, http.StatusOK, apiResponse{Data: item})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing intention id"))
		return
	}

	tag, err := h.pool.Exec(r.Context(),
		`DELETE FROM implementation_intentions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, fmt.Errorf("intention not found"))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"status": "deleted"}})
}
