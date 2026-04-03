package admin

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/auth"
	"joules/internal/config"
)

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

type forceReloader interface {
	ForceReload()
}

type Handler struct {
	pool            *pgxpool.Pool
	requireApproval bool
	cfg             *config.Config
	aiReloader      forceReloader
	srv             *http.Server
}

func NewHandler(pool *pgxpool.Pool, requireApproval bool, cfg *config.Config, aiReloader forceReloader, srv *http.Server) *Handler {
	return &Handler{pool: pool, requireApproval: requireApproval, cfg: cfg, aiReloader: aiReloader, srv: srv}
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
	msg := err.Error()
	if status >= 500 {
		msg = "internal server error"
	}
	writeJSON(w, status, apiResponse{Error: msg})
}

func getSettingDefault(pool *pgxpool.Pool, key, fallback string) string {
	return GetSettingDefault(pool, context.Background(), key, fallback)
}

var serverStartTime = time.Now()

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
	ctx := r.Context()
	var requireApproval string
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'require_approval'").Scan(&requireApproval)

	var aiProvider, aiModel, routingModel, visionModel, ocrModel string
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'ai_provider'").Scan(&aiProvider)
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'ai_model'").Scan(&aiModel)
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'routing_model'").Scan(&routingModel)
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'vision_model'").Scan(&visionModel)
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'ocr_model'").Scan(&ocrModel)

	var customBaseURL, customAPIKey string
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'custom_base_url'").Scan(&customBaseURL)
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'custom_api_key'").Scan(&customAPIKey)

	// SMTP overrides from DB (takes precedence over env)
	var smtpHost, smtpUser, smtpPort string
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'smtp_host'").Scan(&smtpHost)
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'smtp_user'").Scan(&smtpUser)
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'smtp_port'").Scan(&smtpPort)

	// Fall back to config values if not set in DB
	if aiProvider == "" {
		aiProvider = h.cfg.AIProvider
	}
	if aiModel == "" {
		aiModel = h.cfg.AIModel
	}
	if routingModel == "" {
		routingModel = h.cfg.RoutingModel
	}
	if visionModel == "" {
		visionModel = h.cfg.VisionModel
	}
	if ocrModel == "" {
		ocrModel = h.cfg.OCRModel
	}
	if smtpHost == "" {
		smtpHost = h.cfg.SMTPHost
	}
	if smtpUser == "" {
		smtpUser = h.cfg.SMTPUser
	}
	if smtpPort == "" && h.cfg.SMTPPort > 0 {
		smtpPort = fmt.Sprintf("%d", h.cfg.SMTPPort)
	}

	var tavilyAPIKey string
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'tavily_api_key'").Scan(&tavilyAPIKey)

	var ocrProvider string
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'ocr_provider'").Scan(&ocrProvider)
	if ocrProvider == "" {
		ocrProvider = h.cfg.OCRProvider
	}

	smtpConfigured := smtpHost != "" && smtpUser != ""

	// Mask custom API key — show only last 4 chars
	maskedAPIKey := ""
	if customAPIKey != "" && len(customAPIKey) > 4 {
		maskedAPIKey = "****" + customAPIKey[len(customAPIKey)-4:]
	} else if customAPIKey != "" {
		maskedAPIKey = "****"
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"require_approval":          requireApproval == "true",
		"ai_provider":               aiProvider,
		"ai_model":                  aiModel,
		"routing_model":             routingModel,
		"vision_model":              visionModel,
		"ocr_model":                 ocrModel,
		"custom_base_url":           customBaseURL,
		"custom_api_key":            maskedAPIKey,
		"smtp_configured":           smtpConfigured,
		"smtp_host":                 smtpHost,
		"smtp_user":                 smtpUser,
		"smtp_port":                 smtpPort,
		"app_url":                   h.cfg.AppURL,
		"port":                      h.cfg.Port,
		"tavily_api_key_configured": tavilyAPIKey != "",
		"ocr_provider":              ocrProvider,
	}})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequireApproval *bool  `json:"require_approval"`
		AIProvider      string `json:"ai_provider"`
		AIModel         string `json:"ai_model"`
		RoutingModel    string `json:"routing_model"`
		VisionModel     string `json:"vision_model"`
		OCRModel        string `json:"ocr_model"`
		CustomBaseURL   string `json:"custom_base_url"`
		CustomAPIKey    string `json:"custom_api_key"`
		SMTPHost        string `json:"smtp_host"`
		SMTPPort        string `json:"smtp_port"`
		SMTPUser        string `json:"smtp_user"`
		SMTPPass        string `json:"smtp_pass"`
		TavilyAPIKey    string `json:"tavily_api_key"`
		OCRProvider     string `json:"ocr_provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	upsert := func(key, value string) error {
		_, err := h.pool.Exec(r.Context(),
			"INSERT INTO app_settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()",
			key, value,
		)
		return err
	}

	if req.RequireApproval != nil {
		val := "false"
		if *req.RequireApproval {
			val = "true"
		}
		if err := upsert("require_approval", val); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update require_approval: %w", err))
			return
		}
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
	if req.RoutingModel != "" {
		if err := upsert("routing_model", req.RoutingModel); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update routing_model: %w", err))
			return
		}
	}
	if req.VisionModel != "" {
		if err := upsert("vision_model", req.VisionModel); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update vision_model: %w", err))
			return
		}
	}
	if req.OCRModel != "" {
		if err := upsert("ocr_model", req.OCRModel); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update ocr_model: %w", err))
			return
		}
	}
	if req.CustomBaseURL != "" {
		if err := upsert("custom_base_url", req.CustomBaseURL); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update custom_base_url: %w", err))
			return
		}
	}
	if req.CustomAPIKey != "" {
		if err := upsert("custom_api_key", req.CustomAPIKey); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update custom_api_key: %w", err))
			return
		}
	}
	if req.SMTPHost != "" {
		if err := upsert("smtp_host", req.SMTPHost); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update smtp_host: %w", err))
			return
		}
	}
	if req.SMTPPort != "" {
		if err := upsert("smtp_port", req.SMTPPort); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update smtp_port: %w", err))
			return
		}
	}
	if req.SMTPUser != "" {
		if err := upsert("smtp_user", req.SMTPUser); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update smtp_user: %w", err))
			return
		}
	}
	if req.SMTPPass != "" {
		if err := upsert("smtp_pass", req.SMTPPass); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update smtp_pass: %w", err))
			return
		}
	}
	if req.TavilyAPIKey != "" {
		if err := upsert("tavily_api_key", req.TavilyAPIKey); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update tavily_api_key: %w", err))
			return
		}
	}
	if req.OCRProvider != "" {
		if err := upsert("ocr_provider", req.OCRProvider); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update ocr_provider: %w", err))
			return
		}
	}

	if h.aiReloader != nil {
		h.aiReloader.ForceReload()
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"require_approval": req.RequireApproval != nil && *req.RequireApproval,
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.srv.Shutdown(ctx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)
		}
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

// GetFoodsStats returns the count of foods in foods_db and import status.
func (h *Handler) GetFoodsStats(w http.ResponseWriter, r *http.Request) {
	var count int64
	h.pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM foods_db").Scan(&count)

	var status string
	h.pool.QueryRow(r.Context(),
		"SELECT value FROM app_settings WHERE key = 'foods_db_import_status'").Scan(&status)
	if status == "" {
		status = "idle"
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"count":         count,
		"import_status": status,
	}})
}

// ---------- God's Eye ----------

type UserViewProfile struct {
	Name           string   `json:"name"`
	Age            *int32   `json:"age"`
	Sex            *string  `json:"sex"`
	HeightCm       *float64 `json:"height_cm"`
	WeightKg       *float64 `json:"weight_kg"`
	TargetWeightKg *float64 `json:"target_weight_kg"`
	ActivityLevel  *string  `json:"activity_level"`
}

type UserViewGoals struct {
	Objective          string  `json:"objective"`
	DietPlan           string  `json:"diet_plan"`
	FastingWindow      *string `json:"fasting_window"`
	DailyCalorieTarget int32   `json:"daily_calorie_target"`
	DailyProteinG      int32   `json:"daily_protein_g"`
	DailyCarbsG        int32   `json:"daily_carbs_g"`
	DailyFatG          int32   `json:"daily_fat_g"`
}

type UserViewSummary struct {
	TotalCalories int32   `json:"total_calories"`
	TotalProtein  float64 `json:"total_protein"`
	TotalCarbs    float64 `json:"total_carbs"`
	TotalFat      float64 `json:"total_fat"`
	TotalFiber    float64 `json:"total_fiber"`
	TotalBurned   int32   `json:"total_burned"`
	TotalWaterMl  int32   `json:"total_water_ml"`
}

type FoodViewItem struct {
	Name     string  `json:"name"`
	Calories int32   `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	CarbsG   float64 `json:"carbs_g"`
	FatG     float64 `json:"fat_g"`
}

type MealViewItem struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	MealType  string         `json:"meal_type"`
	Note      *string        `json:"note"`
	Foods     []FoodViewItem `json:"foods"`
}

type WeightViewEntry struct {
	Date     string  `json:"date"`
	WeightKg float64 `json:"weight_kg"`
}

type UserViewResponse struct {
	Email         string            `json:"email"`
	CreatedAt     time.Time         `json:"created_at"`
	Date          string            `json:"date"`
	Profile       *UserViewProfile  `json:"profile"`
	Goals         *UserViewGoals    `json:"goals"`
	Summary       UserViewSummary   `json:"summary"`
	Meals         []MealViewItem    `json:"meals"`
	WeightHistory []WeightViewEntry `json:"weight_history"`
}

func (h *Handler) GetUserView(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	ctx := r.Context()

	dateStr := r.URL.Query().Get("date")
	var date time.Time
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date: %w", err))
			return
		}
	} else {
		date = time.Now()
	}

	var resp UserViewResponse
	resp.Date = date.Format("2006-01-02")
	resp.Meals = []MealViewItem{}
	resp.WeightHistory = []WeightViewEntry{}

	// User basic info
	if err := h.pool.QueryRow(ctx, "SELECT email, created_at FROM users WHERE id = $1", targetID).
		Scan(&resp.Email, &resp.CreatedAt); err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("user not found"))
		return
	}

	// Profile (optional — user may not have completed onboarding)
	var profile UserViewProfile
	var heightCm, weightKg, targetWeightKg pgtype.Numeric
	if err := h.pool.QueryRow(ctx,
		"SELECT name, age, sex, height_cm, weight_kg, target_weight_kg, activity_level FROM user_profiles WHERE user_id = $1",
		targetID,
	).Scan(&profile.Name, &profile.Age, &profile.Sex, &heightCm, &weightKg, &targetWeightKg, &profile.ActivityLevel); err == nil {
		if heightCm.Valid {
			f, _ := heightCm.Float64Value()
			v := f.Float64
			profile.HeightCm = &v
		}
		if weightKg.Valid {
			f, _ := weightKg.Float64Value()
			v := f.Float64
			profile.WeightKg = &v
		}
		if targetWeightKg.Valid {
			f, _ := targetWeightKg.Float64Value()
			v := f.Float64
			profile.TargetWeightKg = &v
		}
		resp.Profile = &profile
	}

	// Goals (optional)
	var goals UserViewGoals
	if err := h.pool.QueryRow(ctx,
		"SELECT objective, diet_plan, fasting_window, daily_calorie_target, daily_protein_g, daily_carbs_g, daily_fat_g FROM user_goals WHERE user_id = $1",
		targetID,
	).Scan(&goals.Objective, &goals.DietPlan, &goals.FastingWindow, &goals.DailyCalorieTarget, &goals.DailyProteinG, &goals.DailyCarbsG, &goals.DailyFatG); err == nil {
		resp.Goals = &goals
	}

	// Daily nutrition + water + exercise summary
	const summaryQ = `
		SELECT
			COALESCE(SUM(fi.calories), 0)::int,
			COALESCE(SUM(fi.protein_g), 0)::float8,
			COALESCE(SUM(fi.carbs_g), 0)::float8,
			COALESCE(SUM(fi.fat_g), 0)::float8,
			COALESCE(SUM(fi.fiber_g), 0)::float8,
			COALESCE((SELECT SUM(calories_burned) FROM exercises e WHERE e.user_id = $1 AND e.timestamp::date = $2), 0)::int,
			(SELECT COALESCE(SUM(amount_ml), 0)::int FROM water_logs w WHERE w.user_id = $1 AND w.date = $2)
		FROM meals m
		JOIN food_items fi ON fi.meal_id = m.id
		WHERE m.user_id = $1 AND m.timestamp::date = $2
	`
	// Ignore error — zero values are fine if no data
	h.pool.QueryRow(ctx, summaryQ, targetID, date).Scan(
		&resp.Summary.TotalCalories,
		&resp.Summary.TotalProtein,
		&resp.Summary.TotalCarbs,
		&resp.Summary.TotalFat,
		&resp.Summary.TotalFiber,
		&resp.Summary.TotalBurned,
		&resp.Summary.TotalWaterMl,
	)

	// Meals with food items
	const mealsQ = `
		SELECT m.id, m.timestamp, m.meal_type, m.note,
			fi.id, fi.name, fi.calories, fi.protein_g, fi.carbs_g, fi.fat_g
		FROM meals m
		LEFT JOIN food_items fi ON fi.meal_id = m.id
		WHERE m.user_id = $1 AND m.timestamp::date = $2
		ORDER BY m.timestamp ASC, fi.id ASC
	`
	if rows, err := h.pool.Query(ctx, mealsQ, targetID, date); err == nil {
		defer rows.Close()
		var mealOrder []string
		mealMap := map[string]*MealViewItem{}
		for rows.Next() {
			var (
				mID, mType         string
				mTime              time.Time
				mNote              *string
				fID                *string
				fName              *string
				fCal               *int32
				fProt, fCarb, fFat pgtype.Numeric
			)
			if err := rows.Scan(&mID, &mTime, &mType, &mNote, &fID, &fName, &fCal, &fProt, &fCarb, &fFat); err != nil {
				continue
			}
			if _, ok := mealMap[mID]; !ok {
				mealMap[mID] = &MealViewItem{
					ID:        mID,
					Timestamp: mTime.Format(time.RFC3339),
					MealType:  mType,
					Note:      mNote,
					Foods:     []FoodViewItem{},
				}
				mealOrder = append(mealOrder, mID)
			}
			if fID != nil && fName != nil {
				cal := int32(0)
				if fCal != nil {
					cal = *fCal
				}
				mealMap[mID].Foods = append(mealMap[mID].Foods, FoodViewItem{
					Name:     *fName,
					Calories: cal,
					ProteinG: numericToFloat(fProt),
					CarbsG:   numericToFloat(fCarb),
					FatG:     numericToFloat(fFat),
				})
			}
		}
		resp.Meals = make([]MealViewItem, 0, len(mealOrder))
		for _, id := range mealOrder {
			resp.Meals = append(resp.Meals, *mealMap[id])
		}
	}

	// Weight history (last 30 days)
	if wRows, err := h.pool.Query(ctx,
		"SELECT date, weight_kg FROM weight_logs WHERE user_id = $1 ORDER BY date DESC LIMIT 30",
		targetID,
	); err == nil {
		defer wRows.Close()
		for wRows.Next() {
			var d time.Time
			var kg pgtype.Numeric
			if err := wRows.Scan(&d, &kg); err != nil {
				continue
			}
			resp.WeightHistory = append(resp.WeightHistory, WeightViewEntry{
				Date:     d.Format("2006-01-02"),
				WeightKg: numericToFloat(kg),
			})
		}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *Handler) GetPrompts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result := make(map[string]string, len(DefaultPrompts))
	for key, defaultVal := range DefaultPrompts {
		if val, ok := GetSetting(h.pool, ctx, key); ok && val != "" {
			result[key] = val
		} else {
			result[key] = defaultVal
		}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: result})
}

func (h *Handler) UpdatePrompts(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	for key, text := range req {
		if !strings.HasPrefix(key, "prompt_") {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid prompt key: %s", key))
			return
		}
		if err := UpsertSetting(h.pool, r.Context(), key, text); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update prompt %s: %w", key, err))
			return
		}
	}
	if h.aiReloader != nil {
		h.aiReloader.ForceReload()
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "prompts updated"}})
}

func (h *Handler) GetFeatures(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result := make(map[string]bool, len(DefaultFeatures))
	for name, defaultVal := range DefaultFeatures {
		val, ok := GetSetting(h.pool, ctx, "feature_"+name)
		if ok && val != "" {
			result[name] = val != "false"
		} else {
			result[name] = defaultVal
		}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: result})
}

func (h *Handler) UpdateFeatures(w http.ResponseWriter, r *http.Request) {
	var req map[string]bool
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	for name, enabled := range req {
		val := "false"
		if enabled {
			val = "true"
		}
		if err := UpsertSetting(h.pool, r.Context(), "feature_"+name, val); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update feature %s: %w", name, err))
			return
		}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "features updated"}})
}

func (h *Handler) GetCoachConfig(w http.ResponseWriter, r *http.Request) {
	cfg := GetCoachConfig(h.pool, r.Context())
	writeJSON(w, http.StatusOK, apiResponse{Data: cfg})
}

func (h *Handler) UpdateCoachConfig(w http.ResponseWriter, r *http.Request) {
	var cfg CoachConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("marshal coach config: %w", err))
		return
	}
	if err := UpsertSetting(h.pool, r.Context(), "coach_config", string(data)); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("update coach config: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: cfg})
}

func (h *Handler) GetTDEEConfig(w http.ResponseWriter, r *http.Request) {
	cfg := GetTDEEConfig(h.pool, r.Context())
	writeJSON(w, http.StatusOK, apiResponse{Data: cfg})
}

func (h *Handler) UpdateTDEEConfig(w http.ResponseWriter, r *http.Request) {
	var cfg TDEEConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("marshal tdee config: %w", err))
		return
	}
	if err := UpsertSetting(h.pool, r.Context(), "tdee_config", string(data)); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("update tdee config: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: cfg})
}

func isBlockedHost(host string) bool {
	ips, err := net.LookupIP(host)
	if err != nil {
		return true
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return true
		}
	}
	return false
}

func (h *Handler) GetModels(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	provider := body.Provider
	if provider == "" {
		provider = getSettingDefault(h.pool, "ai_provider", "openai")
	}

	var apiKey, baseURL string
	switch provider {
	case "openai":
		apiKey = h.cfg.OpenAIKey
		baseURL = "https://api.openai.com"
		if h.cfg.OpenAIBaseURL != "" {
			baseURL = h.cfg.OpenAIBaseURL
		}
	case "anthropic":
		apiKey = h.cfg.AnthropicKey
		baseURL = "https://api.anthropic.com"
	case "custom":
		baseURL = getSettingDefault(h.pool, "custom_base_url", "")
		apiKey = getSettingDefault(h.pool, "custom_api_key", "")
		provider = "openai"
	default:
		writeJSON(w, http.StatusBadRequest, apiResponse{Error: "invalid provider"})
		return
	}

	if body.APIKey != "" {
		apiKey = body.APIKey
	}
	if body.BaseURL != "" {
		baseURL = body.BaseURL
	}

	if u, err := url.Parse(baseURL); err == nil {
		if isBlockedHost(u.Hostname()) {
			writeJSON(w, http.StatusBadRequest, apiResponse{Error: "base URL resolves to a blocked address"})
			return
		}
	}

	if apiKey == "" {
		writeJSON(w, http.StatusOK, apiResponse{Data: []interface{}{}})
		return
	}

	type ModelEntry struct {
		ID      string `json:"id"`
		OwnedBy string `json:"owned_by,omitempty"`
		Created int64  `json:"created,omitempty"`
	}

	var models []ModelEntry

	client := &http.Client{Timeout: 15 * time.Second}

	if provider == "anthropic" {
		req, _ := http.NewRequest("GET", baseURL+"/v1/models", nil)
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		resp, err := client.Do(req)
		if err != nil {
			slog.Error("failed to fetch anthropic models", "error", err)
			writeJSON(w, http.StatusOK, apiResponse{Data: []interface{}{}})
			return
		}
		defer resp.Body.Close()
		var result struct {
			Data []struct {
				ID      string `json:"id"`
				Created int64  `json:"created_at"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		for _, m := range result.Data {
			models = append(models, ModelEntry{ID: m.ID, Created: m.Created})
		}
	} else {
		req, _ := http.NewRequest("GET", baseURL+"/v1/models", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := client.Do(req)
		if err != nil {
			slog.Error("failed to fetch models", "error", err)
			writeJSON(w, http.StatusOK, apiResponse{Data: []interface{}{}})
			return
		}
		defer resp.Body.Close()
		var result struct {
			Data []struct {
				ID      string `json:"id"`
				OwnedBy string `json:"owned_by"`
				Created int64  `json:"created"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		for _, m := range result.Data {
			models = append(models, ModelEntry{ID: m.ID, OwnedBy: m.OwnedBy, Created: m.Created})
		}
	}

	if models == nil {
		models = []ModelEntry{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: models})
}

func (h *Handler) TestAI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.Model == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{Error: "model is required"})
		return
	}

	apiKey := req.APIKey
	baseURL := req.BaseURL
	provider := req.Provider

	if apiKey == "" {
		switch provider {
		case "openai":
			apiKey = h.cfg.OpenAIKey
			baseURL = h.cfg.OpenAIBaseURL
		case "anthropic":
			apiKey = h.cfg.AnthropicKey
		case "custom":
			apiKey = getSettingDefault(h.pool, "custom_api_key", "")
			baseURL = getSettingDefault(h.pool, "custom_base_url", "")
		}
	}
	if provider == "custom" {
		if baseURL == "" {
			writeJSON(w, http.StatusBadRequest, apiResponse{Error: "custom provider requires a base URL"})
			return
		}
		provider = "openai"
	}
	if baseURL == "" && provider == "openai" {
		baseURL = "https://api.openai.com"
	} else if baseURL == "" && provider == "anthropic" {
		baseURL = "https://api.anthropic.com"
	}

	if baseURL != "" {
		if u, err := url.Parse(baseURL); err == nil {
			if isBlockedHost(u.Hostname()) {
				writeJSON(w, http.StatusBadRequest, apiResponse{Error: "base URL resolves to a blocked address"})
				return
			}
		}
	}

	type TestResult struct {
		Success         bool   `json:"success"`
		Model           string `json:"model"`
		LatencyMs       int64  `json:"latency_ms"`
		ResponsePreview string `json:"response_preview"`
		Error           string `json:"error,omitempty"`
	}

	start := time.Now()

	var result TestResult
	result.Model = req.Model

	if provider == "anthropic" {
		body := map[string]interface{}{
			"model":      req.Model,
			"max_tokens": 20,
			"messages": []map[string]string{
				{"role": "user", "content": "Say hello in 5 words."},
			},
		}
		bodyBytes, _ := json.Marshal(body)
		httpReq, _ := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", apiKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")

		resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(httpReq)
		if err != nil {
			result.Error = err.Error()
			writeJSON(w, http.StatusOK, apiResponse{Data: result})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errBody)
			result.Error = fmt.Sprintf("API returned status %d: %v", resp.StatusCode, errBody)
			writeJSON(w, http.StatusOK, apiResponse{Data: result})
			return
		}

		var chatResp struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		json.NewDecoder(resp.Body).Decode(&chatResp)
		if len(chatResp.Content) > 0 {
			result.ResponsePreview = chatResp.Content[0].Text
		}
	} else {
		body := map[string]interface{}{
			"model":                 req.Model,
			"max_completion_tokens": 20,
			"messages": []map[string]string{
				{"role": "user", "content": "Say hello in 5 words."},
			},
		}
		bodyBytes, _ := json.Marshal(body)
		httpReq, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(httpReq)
		if err != nil {
			result.Error = err.Error()
			writeJSON(w, http.StatusOK, apiResponse{Data: result})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errBody)
			result.Error = fmt.Sprintf("API returned status %d: %v", resp.StatusCode, errBody)
			writeJSON(w, http.StatusOK, apiResponse{Data: result})
			return
		}

		var chatResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		json.NewDecoder(resp.Body).Decode(&chatResp)
		if len(chatResp.Choices) > 0 {
			result.ResponsePreview = chatResp.Choices[0].Message.Content
		}
	}

	result.LatencyMs = time.Since(start).Milliseconds()
	result.Success = true
	writeJSON(w, http.StatusOK, apiResponse{Data: result})
}

func (h *Handler) GetHealthcheck(w http.ResponseWriter, r *http.Request) {
	type ComponentHealth struct {
		Status    string `json:"status"`
		LatencyMs int64  `json:"latency_ms,omitempty"`
		Error     string `json:"error,omitempty"`
	}

	type HealthResponse struct {
		Postgres      ComponentHealth `json:"postgres"`
		AI            ComponentHealth `json:"ai"`
		UptimeSeconds int64           `json:"uptime_seconds"`
	}

	resp := HealthResponse{
		UptimeSeconds: int64(time.Since(serverStartTime).Seconds()),
	}

	pgStart := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var pgOne int
	err := h.pool.QueryRow(ctx, "SELECT 1").Scan(&pgOne)
	if err != nil {
		resp.Postgres = ComponentHealth{Status: "error", Error: err.Error()}
	} else {
		resp.Postgres = ComponentHealth{Status: "ok", LatencyMs: time.Since(pgStart).Milliseconds()}
	}

	provider := getSettingDefault(h.pool, "ai_provider", "openai")
	hasKey := false
	switch provider {
	case "openai":
		hasKey = h.cfg.OpenAIKey != ""
	case "anthropic":
		hasKey = h.cfg.AnthropicKey != ""
	case "custom":
		hasKey = getSettingDefault(h.pool, "custom_api_key", "") != ""
	}
	if hasKey {
		resp.AI = ComponentHealth{Status: "configured"}
	} else {
		resp.AI = ComponentHealth{Status: "no_api_key", Error: "No API key configured"}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *Handler) GetPublicFeatures(w http.ResponseWriter, r *http.Request) {
	result := make(map[string]bool, len(DefaultFeatures))
	for name, defaultVal := range DefaultFeatures {
		val, ok := GetSetting(h.pool, r.Context(), "feature_"+name)
		if ok && val != "" {
			result[name] = val != "false"
		} else {
			result[name] = defaultVal
		}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: result})
}

func (h *Handler) ImportFoods(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("file too large or invalid form: %w", err))
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing file field: %w", err))
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	var imported, skipped int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if len(record) < 6 {
			skipped++
			continue
		}

		name := strings.TrimSpace(record[0])
		if name == "" || strings.EqualFold(name, "name") {
			skipped++
			continue
		}

		calories, _ := strconv.Atoi(strings.TrimSpace(record[1]))
		protein, _ := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)
		carbs, _ := strconv.ParseFloat(strings.TrimSpace(record[3]), 64)
		fat, _ := strconv.ParseFloat(strings.TrimSpace(record[4]), 64)
		fiber := 0.0
		if len(record) > 5 {
			fiber, _ = strconv.ParseFloat(strings.TrimSpace(record[5]), 64)
		}
		serving := ""
		if len(record) > 6 {
			serving = strings.TrimSpace(record[6])
		}
		barcode := ""
		if len(record) > 7 {
			barcode = strings.TrimSpace(record[7])
		}
		brand := ""
		if len(record) > 8 {
			brand = strings.TrimSpace(record[8])
		}

		_, err = h.pool.Exec(r.Context(),
			`INSERT INTO foods_db (name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, barcode, brand)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (barcode) WHERE barcode IS NOT NULL AND barcode != '' DO NOTHING`,
			name, calories, protein, carbs, fat, fiber, serving, barcode, brand,
		)
		if err != nil {
			skipped++
			continue
		}
		imported++
	}

	_ = UpsertSetting(h.pool, r.Context(), "foods_db_import_status",
		fmt.Sprintf("last_import=%s imported=%d skipped=%d", time.Now().Format("2006-01-02T15:04:05"), imported, skipped))

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"imported": imported,
		"skipped":  skipped,
	}})
}

func (h *Handler) GetNutritionCache(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}

	type CacheEntry struct {
		ID          string  `json:"id"`
		Query       string  `json:"query"`
		Name        string  `json:"name"`
		Calories    int     `json:"calories"`
		ProteinG    float64 `json:"protein_g"`
		CarbsG      float64 `json:"carbs_g"`
		FatG        float64 `json:"fat_g"`
		FiberG      float64 `json:"fiber_g"`
		ServingSize string  `json:"serving_size"`
		Source      string  `json:"source"`
		CreatedAt   string  `json:"created_at"`
	}

	query := `SELECT id, query, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, source, created_at
		 FROM nutrition_cache ORDER BY created_at DESC LIMIT $1`
	args := []any{limit}

	if search != "" {
		query = `SELECT id, query, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, source, created_at
			 FROM nutrition_cache
			 WHERE query ILIKE '%' || $1 || '%' OR name ILIKE '%' || $1 || '%'
			 ORDER BY created_at DESC LIMIT $2`
		args = []any{search, limit}
	}

	dbRows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("query nutrition cache: %w", err))
		return
	}
	defer dbRows.Close()

	var entries []CacheEntry
	for dbRows.Next() {
		var e CacheEntry
		var createdAt time.Time
		if err := dbRows.Scan(&e.ID, &e.Query, &e.Name, &e.Calories, &e.ProteinG, &e.CarbsG, &e.FatG, &e.FiberG, &e.ServingSize, &e.Source, &createdAt); err != nil {
			continue
		}
		e.CreatedAt = createdAt.Format(time.RFC3339)
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []CacheEntry{}
	}

	var count int
	h.pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM nutrition_cache").Scan(&count)

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"entries": entries,
		"total":   count,
	}})
}

func (h *Handler) DeleteNutritionCacheEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := h.pool.Exec(r.Context(), "DELETE FROM nutrition_cache WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete cache entry: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "deleted"}})
}

func (h *Handler) ClearNutritionCache(w http.ResponseWriter, r *http.Request) {
	var count int
	h.pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM nutrition_cache").Scan(&count)
	_, err := h.pool.Exec(r.Context(), "DELETE FROM nutrition_cache")
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("clear cache: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{"deleted": count}})
}
