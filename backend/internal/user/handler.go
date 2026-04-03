package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/admin"
	"joules/internal/auth"
	"joules/internal/config"
	"joules/internal/db/sqlc"
)

type Handler struct {
	q         *sqlc.Queries
	pool      *pgxpool.Pool
	uploadDir string
}

func NewHandler(q *sqlc.Queries, pool *pgxpool.Pool, cfg ...*config.Config) *Handler {
	h := &Handler{q: q, pool: pool, uploadDir: "./uploads"}
	if len(cfg) > 0 && cfg[0] != nil {
		h.uploadDir = cfg[0].UploadDir
	}
	return h
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

func numericToFloatPtr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, _ := n.Float64Value()
	return &f.Float64
}

func intPtr(v int) *int32 {
	i := int32(v)
	return &i
}

func stringPtr(v string) *string {
	return &v
}

func formatPgTime(t pgtype.Time) *string {
	if !t.Valid {
		return nil
	}
	totalMinutes := t.Microseconds / 60_000_000
	hh := totalMinutes / 60
	mm := totalMinutes % 60
	s := fmt.Sprintf("%02d:%02d", hh, mm)
	return &s
}

func (h *Handler) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	var req OnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := validateOnboarding(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	tdeeCfg := admin.GetTDEEConfig(h.pool, r.Context())

	bmr, tdee, calorieTarget, proteinG, carbsG, fatG := CalculateTDEE(
		req.Sex, req.Age, req.WeightKg, req.HeightCm,
		req.ActivityLevel, req.Objective, req.DietPlan, tdeeCfg,
	)

	_, err = h.q.CreateProfile(r.Context(), sqlc.CreateProfileParams{
		UserID:             userID,
		Name:               req.Name,
		Age:                intPtr(req.Age),
		Sex:                stringPtr(req.Sex),
		HeightCm:           floatToNumeric(req.HeightCm),
		WeightKg:           floatToNumeric(req.WeightKg),
		TargetWeightKg:     floatToNumeric(req.TargetWeightKg),
		ActivityLevel:      stringPtr(req.ActivityLevel),
		OnboardingComplete: false,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create profile: %w", err))
		return
	}

	_, err = h.q.CreateGoals(r.Context(), sqlc.CreateGoalsParams{
		UserID:             userID,
		Objective:          req.Objective,
		DietPlan:           req.DietPlan,
		FastingWindow:      req.FastingWindow,
		DailyCalorieTarget: int32(calorieTarget),
		DailyProteinG:      int32(proteinG),
		DailyCarbsG:        int32(carbsG),
		DailyFatG:          int32(fatG),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create goals: %w", err))
		return
	}

	if req.DietPlan == "intermittent_fasting" && req.EatingWindowStart != nil && *req.EatingWindowStart != "" {
		parts := strings.Split(*req.EatingWindowStart, ":")
		if len(parts) == 2 {
			var hh, mm int
			fmt.Sscanf(*req.EatingWindowStart, "%d:%d", &hh, &mm)
			pgTime := pgtype.Time{
				Microseconds: int64(hh)*3600_000_000 + int64(mm)*60_000_000,
				Valid:        true,
			}
			_ = h.q.UpdateEatingWindow(r.Context(), sqlc.UpdateEatingWindowParams{
				UserID:            userID,
				EatingWindowStart: pgTime,
			})
		}
	}

	if err := h.q.CompleteOnboarding(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("complete onboarding: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: OnboardingResponse{
		BMR:                bmr,
		TDEE:               tdee,
		CalorieTarget:      int32(calorieTarget),
		ProteinG:           int32(proteinG),
		CarbsG:             int32(carbsG),
		FatG:               int32(fatG),
		OnboardingComplete: true,
	}})
}

func (h *Handler) getAvatarURL(r *http.Request, userID string) *string {
	var avatarURL *string
	h.pool.QueryRow(r.Context(), "SELECT avatar_url FROM user_profiles WHERE user_id = $1", userID).Scan(&avatarURL)
	return avatarURL
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	profile, err := h.q.GetProfile(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("profile not found: %w", err))
		return
	}

	var isAdmin bool
	h.pool.QueryRow(r.Context(), "SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)

	writeJSON(w, http.StatusOK, apiResponse{Data: ProfileResponse{
		Name:               profile.Name,
		Age:                profile.Age,
		Sex:                profile.Sex,
		HeightCm:           numericToFloatPtr(profile.HeightCm),
		WeightKg:           numericToFloatPtr(profile.WeightKg),
		TargetWeightKg:     numericToFloatPtr(profile.TargetWeightKg),
		ActivityLevel:      profile.ActivityLevel,
		OnboardingComplete: profile.OnboardingComplete,
		IsAdmin:            isAdmin,
		AvatarURL:          h.getAvatarURL(r, userID),
	}})
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := validateUpdateProfile(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	err = h.q.UpdateProfile(r.Context(), sqlc.UpdateProfileParams{
		UserID:         userID,
		Name:           req.Name,
		Age:            intPtr(req.Age),
		Sex:            stringPtr(req.Sex),
		HeightCm:       floatToNumeric(req.HeightCm),
		WeightKg:       floatToNumeric(req.WeightKg),
		TargetWeightKg: floatToNumeric(req.TargetWeightKg),
		ActivityLevel:  stringPtr(req.ActivityLevel),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("update profile: %w", err))
		return
	}

	profile, err := h.q.GetProfile(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get profile after update: %w", err))
		return
	}

	heightCm := 0.0
	if f := numericToFloatPtr(profile.HeightCm); f != nil {
		heightCm = *f
	}
	weightKg := 0.0
	if f := numericToFloatPtr(profile.WeightKg); f != nil {
		weightKg = *f
	}
	age := 25
	if profile.Age != nil {
		age = int(*profile.Age)
	}
	sex := "male"
	if profile.Sex != nil {
		sex = *profile.Sex
	}
	activityLevel := "sedentary"
	if profile.ActivityLevel != nil {
		activityLevel = *profile.ActivityLevel
	}

	goals, err := h.q.GetGoals(r.Context(), userID)
	if err == nil {
		tdeeCfg := admin.GetTDEEConfig(h.pool, r.Context())

		_, _, calorieTarget, proteinG, carbsG, fatG := CalculateTDEE(
			sex, age, weightKg, heightCm,
			activityLevel, goals.Objective, goals.DietPlan, tdeeCfg,
		)

		h.q.CreateGoals(r.Context(), sqlc.CreateGoalsParams{
			UserID:             userID,
			Objective:          goals.Objective,
			DietPlan:           goals.DietPlan,
			FastingWindow:      goals.FastingWindow,
			DailyCalorieTarget: int32(calorieTarget),
			DailyProteinG:      int32(proteinG),
			DailyCarbsG:        int32(carbsG),
			DailyFatG:          int32(fatG),
		})
	}

	var isAdmin bool
	h.pool.QueryRow(r.Context(), "SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)

	writeJSON(w, http.StatusOK, apiResponse{Data: ProfileResponse{
		Name:               profile.Name,
		Age:                profile.Age,
		Sex:                profile.Sex,
		HeightCm:           numericToFloatPtr(profile.HeightCm),
		WeightKg:           numericToFloatPtr(profile.WeightKg),
		TargetWeightKg:     numericToFloatPtr(profile.TargetWeightKg),
		ActivityLevel:      profile.ActivityLevel,
		OnboardingComplete: profile.OnboardingComplete,
		IsAdmin:            isAdmin,
		AvatarURL:          h.getAvatarURL(r, userID),
	}})
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("file too large or invalid form: %w", err))
		return
	}

	file, _, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing avatar field: %w", err))
		return
	}
	defer file.Close()

	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("failed to read file header"))
		return
	}
	contentType := http.DetectContentType(head[:n])
	if !strings.HasPrefix(contentType, "image/") {
		writeError(w, http.StatusBadRequest, fmt.Errorf("file must be an image, got %s", contentType))
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("seek failed"))
		return
	}

	avatarDir := filepath.Join(h.uploadDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create avatar dir: %w", err))
		return
	}

	destPath := filepath.Join(avatarDir, userID+".jpg")
	dest, err := os.Create(destPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create avatar file: %w", err))
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save avatar: %w", err))
		return
	}

	avatarURL := "/uploads/avatars/" + userID + ".jpg"
	_, err = h.pool.Exec(r.Context(),
		"UPDATE user_profiles SET avatar_url = $1, updated_at = NOW() WHERE user_id = $2",
		avatarURL, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("update avatar_url: %w", err))
		return
	}

	slog.Info("avatar uploaded", "user_id", userID)
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"avatar_url": avatarURL}})
}

func (h *Handler) GetGoals(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	goals, err := h.q.GetGoals(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("goals not found: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: GoalsResponse{
		Objective:          goals.Objective,
		DietPlan:           goals.DietPlan,
		FastingWindow:      goals.FastingWindow,
		DailyCalorieTarget: goals.DailyCalorieTarget,
		DailyProteinG:      goals.DailyProteinG,
		DailyCarbsG:        goals.DailyCarbsG,
		DailyFatG:          goals.DailyFatG,
		EatingWindowStart:  formatPgTime(goals.EatingWindowStart),
		FastingStreak:      goals.FastingStreak,
	}})
}

func (h *Handler) UpdateGoals(w http.ResponseWriter, r *http.Request) {
	var req UpdateGoalsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if _, ok := objectiveMultipliers[req.Objective]; !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid objective"))
		return
	}
	if _, ok := macroSplits[req.DietPlan]; !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid diet plan"))
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	ctx := r.Context()

	var calTarget, proteinG, carbsG, fatG int32

	if req.ManualOverride {
		calTarget = req.DailyCalorieTarget
		proteinG = req.DailyProteinG
		carbsG = req.DailyCarbsG
		fatG = req.DailyFatG
	} else {
		profile, err := h.q.GetProfile(ctx, userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("get profile: %w", err))
			return
		}

		weightKg, heightCm := 70.0, 170.0
		if f := numericToFloatPtr(profile.WeightKg); f != nil {
			weightKg = *f
		}
		if f := numericToFloatPtr(profile.HeightCm); f != nil {
			heightCm = *f
		}
		age := 25
		if profile.Age != nil {
			age = int(*profile.Age)
		}
		sex := "male"
		if profile.Sex != nil {
			sex = *profile.Sex
		}
		activityLevel := "sedentary"
		if profile.ActivityLevel != nil {
			activityLevel = *profile.ActivityLevel
		}

		tdeeCfg := admin.GetTDEEConfig(h.pool, ctx)

		_, _, cal, prot, carbs, fat := CalculateTDEE(sex, age, weightKg, heightCm, activityLevel, req.Objective, req.DietPlan, tdeeCfg)
		calTarget = int32(cal)
		proteinG = int32(prot)
		carbsG = int32(carbs)
		fatG = int32(fat)
	}

	goals, err := h.q.CreateGoals(ctx, sqlc.CreateGoalsParams{
		UserID:             userID,
		Objective:          req.Objective,
		DietPlan:           req.DietPlan,
		FastingWindow:      req.FastingWindow,
		DailyCalorieTarget: calTarget,
		DailyProteinG:      proteinG,
		DailyCarbsG:        carbsG,
		DailyFatG:          fatG,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("update goals: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: GoalsResponse{
		Objective:          goals.Objective,
		DietPlan:           goals.DietPlan,
		FastingWindow:      goals.FastingWindow,
		DailyCalorieTarget: goals.DailyCalorieTarget,
		DailyProteinG:      goals.DailyProteinG,
		DailyCarbsG:        goals.DailyCarbsG,
		DailyFatG:          goals.DailyFatG,
		EatingWindowStart:  formatPgTime(goals.EatingWindowStart),
		FastingStreak:      goals.FastingStreak,
	}})
}

type PreferencesRequest struct {
	DietType      string   `json:"diet_type"`
	Allergies     []string `json:"allergies"`
	FoodNotes     string   `json:"food_notes"`
	EatingContext string   `json:"eating_context"`
	HeightUnit    string   `json:"height_unit"`
	WeightUnit    string   `json:"weight_unit"`
	EnergyUnit    string   `json:"energy_unit"`
}

type PreferencesResponse struct {
	DietType      string   `json:"diet_type"`
	Allergies     []string `json:"allergies"`
	FoodNotes     string   `json:"food_notes"`
	EatingContext string   `json:"eating_context"`
	HeightUnit    string   `json:"height_unit"`
	WeightUnit    string   `json:"weight_unit"`
	EnergyUnit    string   `json:"energy_unit"`
}

func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var prefs PreferencesResponse
	err = h.pool.QueryRow(r.Context(),
		"SELECT diet_type, allergies, food_notes, eating_context, height_unit, weight_unit, energy_unit FROM user_preferences WHERE user_id = $1",
		userID,
	).Scan(&prefs.DietType, &prefs.Allergies, &prefs.FoodNotes, &prefs.EatingContext, &prefs.HeightUnit, &prefs.WeightUnit, &prefs.EnergyUnit)
	if err != nil {
		// Return defaults if not set
		prefs = PreferencesResponse{DietType: "omnivore", Allergies: []string{}, HeightUnit: "cm", WeightUnit: "kg", EnergyUnit: "kcal"}
	}
	if prefs.Allergies == nil {
		prefs.Allergies = []string{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: prefs})
}

func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var req PreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Allergies == nil {
		req.Allergies = []string{}
	}
	if req.HeightUnit == "" {
		req.HeightUnit = "cm"
	}
	if req.WeightUnit == "" {
		req.WeightUnit = "kg"
	}
	if req.EnergyUnit == "" {
		req.EnergyUnit = "kcal"
	}
	_, err = h.pool.Exec(r.Context(),
		`INSERT INTO user_preferences (user_id, diet_type, allergies, food_notes, eating_context, height_unit, weight_unit, energy_unit)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (user_id) DO UPDATE SET
		   diet_type = $2, allergies = $3, food_notes = $4, eating_context = $5,
		   height_unit = $6, weight_unit = $7, energy_unit = $8, updated_at = NOW()`,
		userID, req.DietType, req.Allergies, req.FoodNotes, req.EatingContext,
		req.HeightUnit, req.WeightUnit, req.EnergyUnit,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save preferences: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: PreferencesResponse{
		DietType:      req.DietType,
		Allergies:     req.Allergies,
		FoodNotes:     req.FoodNotes,
		EatingContext: req.EatingContext,
		HeightUnit:    req.HeightUnit,
		WeightUnit:    req.WeightUnit,
		EnergyUnit:    req.EnergyUnit,
	}})
}

func validateOnboarding(req *OnboardingRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.Sex != "male" && req.Sex != "female" {
		return errors.New("sex must be 'male' or 'female'")
	}
	if req.Age < 1 || req.Age > 150 {
		return errors.New("age must be between 1 and 150")
	}
	if req.WeightKg <= 0 {
		return errors.New("weight must be greater than 0")
	}
	if req.HeightCm <= 0 {
		return errors.New("height must be greater than 0")
	}
	if _, ok := activityMultipliers[req.ActivityLevel]; !ok {
		return errors.New("invalid activity level")
	}
	if _, ok := objectiveMultipliers[req.Objective]; !ok {
		return errors.New("invalid objective")
	}
	if _, ok := macroSplits[req.DietPlan]; !ok {
		return errors.New("invalid diet plan")
	}
	return nil
}

func validateUpdateProfile(req *UpdateProfileRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.Sex != "male" && req.Sex != "female" {
		return errors.New("sex must be 'male' or 'female'")
	}
	if req.Age < 1 || req.Age > 150 {
		return errors.New("age must be between 1 and 150")
	}
	if req.WeightKg <= 0 {
		return errors.New("weight must be greater than 0")
	}
	if req.HeightCm <= 0 {
		return errors.New("height must be greater than 0")
	}
	if _, ok := activityMultipliers[req.ActivityLevel]; !ok {
		return errors.New("invalid activity level")
	}
	return nil
}

func (h *Handler) GetCoachNotes(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var notes string
	err = h.pool.QueryRow(r.Context(),
		"SELECT COALESCE(coach_notes, '') FROM user_profiles WHERE user_id = $1", userID,
	).Scan(&notes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"notes": notes}})
}

func (h *Handler) UpdateCoachNotes(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Notes) > 10000 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("notes must be 10000 characters or less"))
		return
	}
	_, err = h.pool.Exec(r.Context(),
		"UPDATE user_profiles SET coach_notes = $1 WHERE user_id = $2", req.Notes, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"notes": req.Notes}})
}

func (h *Handler) GetCoachMemories(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	rows, err := h.pool.Query(r.Context(),
		"SELECT id, category, content, source, created_at, updated_at FROM coach_memory WHERE user_id = $1 ORDER BY category, created_at DESC",
		userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	type Memory struct {
		ID        string `json:"id"`
		Category  string `json:"category"`
		Content   string `json:"content"`
		Source    string `json:"source"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}
	var memories []Memory
	for rows.Next() {
		var m Memory
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&m.ID, &m.Category, &m.Content, &m.Source, &createdAt, &updatedAt); err != nil {
			continue
		}
		m.CreatedAt = createdAt.Format(time.RFC3339)
		m.UpdatedAt = updatedAt.Format(time.RFC3339)
		memories = append(memories, m)
	}
	if memories == nil {
		memories = []Memory{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: memories})
}

func (h *Handler) DeleteCoachMemory(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	memoryID := chi.URLParam(r, "id")
	if memoryID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("memory id required"))
		return
	}
	tag, err := h.pool.Exec(r.Context(),
		"DELETE FROM coach_memory WHERE id = $1 AND user_id = $2", memoryID, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, fmt.Errorf("memory not found"))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"status": "deleted"}})
}
