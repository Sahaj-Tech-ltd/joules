package notify

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"joules/internal/auth"
	"joules/internal/config"
	"joules/internal/db/sqlc"
)

type Handler struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
	svc  *Service
	cfg  *config.Config
}

func NewHandler(q *sqlc.Queries, pool *pgxpool.Pool, svc *Service, cfg *config.Config) *Handler {
	return &Handler{q: q, pool: pool, svc: svc, cfg: cfg}
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
	slog.Error("notify request error", "status", status, "error", err)
	msg := err.Error()
	if status >= 500 {
		msg = "internal server error"
	}
	writeJSON(w, status, apiResponse{Error: msg})
}

func getUserID(r *http.Request) (string, error) {
	uid, ok := r.Context().Value(auth.ContextUserID).(string)
	if !ok {
		return "", fmt.Errorf("unauthorized")
	}
	return uid, nil
}

// GetVAPIDPublicKey returns the VAPID public key so the frontend can subscribe.
func (h *Handler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{
		"public_key": h.cfg.VAPIDPublicKey,
	}})
}

type subscribeRequest struct {
	Endpoint  string `json:"endpoint"`
	P256dh    string `json:"p256dh"`
	Auth      string `json:"auth"`
	UserAgent string `json:"user_agent"`
}

// Subscribe saves a browser push subscription for the current user.
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Endpoint == "" || req.P256dh == "" || req.Auth == "" {
		writeError(w, http.StatusBadRequest, errors.New("endpoint, p256dh, and auth are required"))
		return
	}

	uid, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if err := h.q.SavePushSubscription(r.Context(), sqlc.SavePushSubscriptionParams{
		UserID:    uid,
		Endpoint:  req.Endpoint,
		P256dh:    req.P256dh,
		Auth:      req.Auth,
		UserAgent: req.UserAgent,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Auto-generate a personal ntfy topic for this user if they don't have one yet.
	// This means zero manual setup — just tap "Subscribe in ntfy app" on the frontend.
	ntfyTopic := ""
	prefs, err := h.q.GetNotificationPrefs(r.Context(), uid)
	if err != nil || prefs.NtfyTopic == "" {
		ntfyTopic = "joules-" + uuid.New().String()[:8]
		_, _ = h.q.UpsertNotificationPrefs(r.Context(), sqlc.UpsertNotificationPrefsParams{
			UserID:             uid,
			WaterReminders:     true,
			WaterIntervalHours: 2,
			MealReminders:      true,
			IfWindowReminders:  true,
			StreakReminders:    true,
			QuietStart:         22,
			QuietEnd:           8,
			NtfyTopic:          ntfyTopic,
		})
	} else {
		ntfyTopic = prefs.NtfyTopic
	}

	ntfyURL := ""
	if h.cfg.NtfyBaseURL != "" && ntfyTopic != "" {
		ntfyURL = fmt.Sprintf("%s/%s", h.cfg.NtfyBaseURL, ntfyTopic)
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{
		"status":     "subscribed",
		"ntfy_url":   ntfyURL,
		"ntfy_topic": ntfyTopic,
	}})
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

// Unsubscribe removes a push subscription.
func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var req unsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	uid, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	if err := h.q.DeletePushSubscription(r.Context(), sqlc.DeletePushSubscriptionParams{
		Endpoint: req.Endpoint,
		UserID:   uid,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"status": "unsubscribed"}})
}

// GetPreferences returns the user's notification preferences (or sensible defaults).
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	uid, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	prefs, err := h.q.GetNotificationPrefs(r.Context(), uid)
	if err != nil {
		// Return defaults if not yet set
		writeJSON(w, http.StatusOK, apiResponse{Data: sqlc.NotificationPreference{
			WaterReminders:     true,
			WaterIntervalHours: 2,
			MealReminders:      true,
			IfWindowReminders:  true,
			StreakReminders:    true,
			QuietStart:         22,
			QuietEnd:           8,
		}})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: prefs})
}

type prefsRequest struct {
	WaterReminders     bool   `json:"water_reminders"`
	WaterIntervalHours int32  `json:"water_interval_hours"`
	MealReminders      bool   `json:"meal_reminders"`
	IfWindowReminders  bool   `json:"if_window_reminders"`
	StreakReminders    bool   `json:"streak_reminders"`
	QuietStart         int32  `json:"quiet_start"`
	QuietEnd           int32  `json:"quiet_end"`
	NtfyTopic          string `json:"ntfy_topic"`
}

// SavePreferences upserts the user's notification preferences.
func (h *Handler) SavePreferences(w http.ResponseWriter, r *http.Request) {
	var req prefsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.WaterIntervalHours < 1 {
		req.WaterIntervalHours = 2
	}
	if req.QuietStart < 0 || req.QuietStart > 23 {
		req.QuietStart = 22
	}
	if req.QuietEnd < 0 || req.QuietEnd > 23 {
		req.QuietEnd = 8
	}

	uid, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	prefs, err := h.q.UpsertNotificationPrefs(r.Context(), sqlc.UpsertNotificationPrefsParams{
		UserID:             uid,
		WaterReminders:     req.WaterReminders,
		WaterIntervalHours: req.WaterIntervalHours,
		MealReminders:      req.MealReminders,
		IfWindowReminders:  req.IfWindowReminders,
		StreakReminders:    req.StreakReminders,
		QuietStart:         req.QuietStart,
		QuietEnd:           req.QuietEnd,
		NtfyTopic:          req.NtfyTopic,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: prefs})
}

// SendTest sends a test notification to verify the setup works.
func (h *Handler) SendTest(w http.ResponseWriter, r *http.Request) {
	uid, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	h.svc.SendToUser(r.Context(), uid, Payload{
		Title: "Joules Notifications ✓",
		Body:  "Notifications are working! You'll receive water, meal, and goal reminders here.",
		URL:   "/dashboard",
		Tag:   "test",
	})
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"status": "sent"}})
}

type registerExpoPushReq struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

func (h *Handler) RegisterExpoPush(w http.ResponseWriter, r *http.Request) {
	uid, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	var req registerExpoPushReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, errors.New("token is required"))
		return
	}
	if req.Platform != "ios" && req.Platform != "android" {
		writeError(w, http.StatusBadRequest, errors.New("platform must be 'ios' or 'android'"))
		return
	}

	_, err = h.pool.Exec(r.Context(),
		`INSERT INTO expo_push_tokens (user_id, token, platform) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, token) DO UPDATE SET platform = $3`,
		uid, req.Token, req.Platform)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"status": "registered"}})
}
