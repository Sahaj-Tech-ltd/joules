package admin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"joule/internal/auth"
	"joule/internal/config"
)

type Handler struct {
	pool            *pgxpool.Pool
	requireApproval bool
	cfg             *config.Config
}

func NewHandler(pool *pgxpool.Pool, requireApproval bool, cfg *config.Config) *Handler {
	return &Handler{pool: pool, requireApproval: requireApproval, cfg: cfg}
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
	slog.Error("admin request error", "status", status, "error", err)
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

type UserRow struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Verified  bool      `json:"verified"`
	Approved  bool      `json:"approved"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, email, verified, approved, is_admin, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("query users: %w", err))
		return
	}
	defer rows.Close()

	var users []UserRow
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.Verified, &u.Approved, &u.IsAdmin, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}
	if users == nil {
		users = []UserRow{}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: users})
}

func (h *Handler) ApproveUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerID, _ := r.Context().Value(auth.ContextUserID).(string)
	if id == callerID {
		writeError(w, http.StatusBadRequest, fmt.Errorf("cannot modify your own approval status"))
		return
	}
	_, err := h.pool.Exec(r.Context(), "UPDATE users SET approved = TRUE WHERE id = $1 AND is_admin = FALSE", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("approve user: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "user approved"}})
}

func (h *Handler) UnapproveUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerID, _ := r.Context().Value(auth.ContextUserID).(string)
	if id == callerID {
		writeError(w, http.StatusBadRequest, fmt.Errorf("cannot modify your own approval status"))
		return
	}
	_, err := h.pool.Exec(r.Context(), "UPDATE users SET approved = FALSE WHERE id = $1 AND is_admin = FALSE", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("unapprove user: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "user unapproved"}})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerID, _ := r.Context().Value(auth.ContextUserID).(string)
	if id == callerID {
		writeError(w, http.StatusBadRequest, fmt.Errorf("cannot delete your own account"))
		return
	}
	_, err := h.pool.Exec(r.Context(), "DELETE FROM users WHERE id = $1 AND is_admin = FALSE", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete user: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "user deleted"}})
}

func (h *Handler) MakeAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerID, _ := r.Context().Value(auth.ContextUserID).(string)
	if id == callerID {
		writeError(w, http.StatusBadRequest, fmt.Errorf("cannot modify your own admin status"))
		return
	}
	_, err := h.pool.Exec(r.Context(), "UPDATE users SET is_admin = TRUE, approved = TRUE WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("make admin: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "user promoted to admin"}})
}

func (h *Handler) RemoveAdmin(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerID, _ := r.Context().Value(auth.ContextUserID).(string)
	if id == callerID {
		writeError(w, http.StatusBadRequest, fmt.Errorf("cannot modify your own admin status"))
		return
	}
	_, err := h.pool.Exec(r.Context(), "UPDATE users SET is_admin = FALSE WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("remove admin: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "admin rights removed"}})
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	var requireApproval string
	h.pool.QueryRow(r.Context(), "SELECT value FROM app_settings WHERE key = 'require_approval'").Scan(&requireApproval)

	var aiProvider, aiModel string
	h.pool.QueryRow(r.Context(), "SELECT value FROM app_settings WHERE key = 'ai_provider'").Scan(&aiProvider)
	h.pool.QueryRow(r.Context(), "SELECT value FROM app_settings WHERE key = 'ai_model'").Scan(&aiModel)

	// Fall back to config values if not set in DB
	if aiProvider == "" {
		aiProvider = h.cfg.AIProvider
	}
	if aiModel == "" {
		aiModel = h.cfg.AIModel
	}

	smtpConfigured := h.cfg.SMTPHost != "" && h.cfg.SMTPUser != "" && h.cfg.SMTPPass != ""

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"require_approval": requireApproval == "true",
		"ai_provider":      aiProvider,
		"ai_model":         aiModel,
		"smtp_configured":  smtpConfigured,
		"app_url":          h.cfg.AppURL,
		"port":             h.cfg.Port,
	}})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequireApproval bool   `json:"require_approval"`
		AIProvider      string `json:"ai_provider"`
		AIModel         string `json:"ai_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	val := "false"
	if req.RequireApproval {
		val = "true"
	}
	upsert := func(key, value string) error {
		_, err := h.pool.Exec(r.Context(),
			"INSERT INTO app_settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()",
			key, value,
		)
		return err
	}

	if err := upsert("require_approval", val); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("update require_approval: %w", err))
		return
	}
	if req.AIProvider != "" {
		if err := upsert("ai_provider", req.AIProvider); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update ai_provider: %w", err))
			return
		}
	}
	if req.AIModel != "" {
		if err := upsert("ai_model", req.AIModel); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update ai_model: %w", err))
			return
		}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"require_approval": req.RequireApproval,
		"ai_provider":      req.AIProvider,
		"ai_model":         req.AIModel,
	}})
}

func (h *Handler) GetInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dbSettings := map[string]string{
		"require_approval": "false",
		"ai_provider":      "",
		"ai_model":         "",
	}

	rows, err := h.pool.Query(ctx,
		"SELECT key, value FROM app_settings WHERE key IN ('require_approval', 'ai_provider', 'ai_model')",
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("query app_settings: %w", err))
		return
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		dbSettings[k] = v
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"require_approval": dbSettings["require_approval"] == "true",
		"ai_provider":      dbSettings["ai_provider"],
		"ai_model":         dbSettings["ai_model"],
	}})
}

func (h *Handler) RestartServer(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "restarting server"}})
	go func() {
		slog.Info("admin triggered server restart")
		os.Exit(0)
	}()
}

func (h *Handler) VerifyUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerID, _ := r.Context().Value(auth.ContextUserID).(string)
	if id == callerID {
		writeError(w, http.StatusBadRequest, fmt.Errorf("cannot modify your own account"))
		return
	}
	_, err := h.pool.Exec(r.Context(), "UPDATE users SET verified = TRUE WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("verify user: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "email verified"}})
}

type BannerRow struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Type      string     `json:"type"`
	Active    bool       `json:"active"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (h *Handler) GetBanners(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, title, message, type, active, expires_at, created_at
		 FROM admin_banners
		 WHERE active = TRUE AND (expires_at IS NULL OR expires_at > NOW())
		 ORDER BY created_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("query banners: %w", err))
		return
	}
	defer rows.Close()

	var banners []BannerRow
	for rows.Next() {
		var b BannerRow
		if err := rows.Scan(&b.ID, &b.Title, &b.Message, &b.Type, &b.Active, &b.ExpiresAt, &b.CreatedAt); err != nil {
			continue
		}
		banners = append(banners, b)
	}
	if banners == nil {
		banners = []BannerRow{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: banners})
}

func (h *Handler) CreateBanner(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title     string     `json:"title"`
		Message   string     `json:"message"`
		Type      string     `json:"type"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("message is required"))
		return
	}
	if req.Type == "" {
		req.Type = "info"
	}
	var b BannerRow
	err := h.pool.QueryRow(r.Context(),
		`INSERT INTO admin_banners (title, message, type, expires_at) VALUES ($1, $2, $3, $4)
		 RETURNING id, title, message, type, active, expires_at, created_at`,
		req.Title, req.Message, req.Type, req.ExpiresAt,
	).Scan(&b.ID, &b.Title, &b.Message, &b.Type, &b.Active, &b.ExpiresAt, &b.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create banner: %w", err))
		return
	}
	writeJSON(w, http.StatusCreated, apiResponse{Data: b})
}

func (h *Handler) DeleteBanner(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := h.pool.Exec(r.Context(), "DELETE FROM admin_banners WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete banner: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "banner deleted"}})
}

type LogRow struct {
	ID        int64           `json:"id"`
	Level     string          `json:"level"`
	Category  string          `json:"category"`
	Message   string          `json:"message"`
	Details   json.RawMessage `json:"details,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

func (h *Handler) GetLogs(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	level := r.URL.Query().Get("level")

	query := `SELECT id, level, category, message, details, created_at FROM system_logs WHERE 1=1`
	args := []any{}
	i := 1
	if category != "" && category != "all" {
		query += fmt.Sprintf(" AND category = $%d", i)
		args = append(args, category)
		i++
	}
	if level != "" && level != "all" {
		query += fmt.Sprintf(" AND level = $%d", i)
		args = append(args, level)
		i++
	}
	query += " ORDER BY created_at DESC LIMIT 200"

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("query logs: %w", err))
		return
	}
	defer rows.Close()

	var logs []LogRow
	for rows.Next() {
		var l LogRow
		if err := rows.Scan(&l.ID, &l.Level, &l.Category, &l.Message, &l.Details, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []LogRow{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: logs})
}
