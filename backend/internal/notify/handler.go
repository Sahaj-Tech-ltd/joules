package notify

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"fmt"
	"joule/internal/auth"
	"github.com/google/uuid"
	"joule/internal/config"
	"joule/internal/db/sqlc"
)

type Handler struct {
	q    *sqlc.Queries
	svc  *Service
	cfg  *config.Config
}

func NewHandler(q *sqlc.Queries, svc *Service, cfg *config.Config) *Handler {
	return &Handler{q: q, svc: svc, cfg: cfg}
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
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

func userID(r *http.Request) string {
	return r.Context().Value(auth.ContextUserID).(string)
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

	uid := userID(r)
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
			NtfyTopic:         ntfyTopic,
		})
	} else {
		ntfyTopic = prefs.NtfyTopic
	}

	ntfyURL := ""
	if h.cfg.NtfyBaseURL != "" && ntfyTopic != "" {
		ntfyURL = fmt.Sprintf("%s/%s", h.cfg.NtfyBaseURL, ntfyTopic)
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{
		"status":    "subscribed",
		"ntfy_url":  ntfyURL,
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

	if err := h.q.DeletePushSubscription(r.Context(), sqlc.DeletePushSubscriptionParams{
		Endpoint: req.Endpoint,
		UserID:   userID(r),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"status": "unsubscribed"}})
}

// GetPreferences returns the user's notification preferences (or sensible defaults).
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	prefs, err := h.q.GetNotificationPrefs(r.Context(), userID(r))
	if err != nil {
		// Return defaults if not yet set
		writeJSON(w, http.StatusOK, apiResponse{Data: sqlc.NotificationPreferences{
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
	NtfyTopic         string `json:"ntfy_topic"`
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

	prefs, err := h.q.UpsertNotificationPrefs(r.Context(), sqlc.UpsertNotificationPrefsParams{
		UserID:             userID(r),
		WaterReminders:     req.WaterReminders,
		WaterIntervalHours: req.WaterIntervalHours,
		MealReminders:      req.MealReminders,
		IfWindowReminders:  req.IfWindowReminders,
		StreakReminders:    req.StreakReminders,
		QuietStart:         req.QuietStart,
		QuietEnd:           req.QuietEnd,
		NtfyTopic:         req.NtfyTopic,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: prefs})
}

// SendTest sends a test notification to verify the setup works.
func (h *Handler) SendTest(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	h.svc.SendToUser(r.Context(), uid, Payload{
		Title: "Joules Notifications ✓",
		Body:  "Notifications are working! You'll receive water, meal, and goal reminders here.",
		URL:   "/dashboard",
		Tag:   "test",
	})
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"status": "sent"}})
}
