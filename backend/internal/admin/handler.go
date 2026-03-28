package admin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joule/internal/auth"
	"joule/internal/config"
)

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

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
	ctx := r.Context()
	var requireApproval string
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'require_approval'").Scan(&requireApproval)

	var aiProvider, aiModel, routingModel string
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'ai_provider'").Scan(&aiProvider)
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'ai_model'").Scan(&aiModel)
	h.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = 'routing_model'").Scan(&routingModel)

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
	if smtpHost == "" {
		smtpHost = h.cfg.SMTPHost
	}
	if smtpUser == "" {
		smtpUser = h.cfg.SMTPUser
	}
	if smtpPort == "" && h.cfg.SMTPPort > 0 {
		smtpPort = fmt.Sprintf("%d", h.cfg.SMTPPort)
	}

	smtpConfigured := smtpHost != "" && smtpUser != ""

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"require_approval": requireApproval == "true",
		"ai_provider":      aiProvider,
		"ai_model":         aiModel,
		"routing_model":    routingModel,
		"smtp_configured":  smtpConfigured,
		"smtp_host":        smtpHost,
		"smtp_user":        smtpUser,
		"smtp_port":        smtpPort,
		"app_url":          h.cfg.AppURL,
		"port":             h.cfg.Port,
	}})
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequireApproval bool   `json:"require_approval"`
		AIProvider      string `json:"ai_provider"`
		AIModel         string `json:"ai_model"`
		RoutingModel    string `json:"routing_model"`
		SMTPHost        string `json:"smtp_host"`
		SMTPPort        string `json:"smtp_port"`
		SMTPUser        string `json:"smtp_user"`
		SMTPPass        string `json:"smtp_pass"`
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
	if req.RoutingModel != "" {
		if err := upsert("routing_model", req.RoutingModel); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("update routing_model: %w", err))
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
				mID, mType string
				mTime      time.Time
				mNote      *string
				fID        *string
				fName      *string
				fCal       *int32
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
