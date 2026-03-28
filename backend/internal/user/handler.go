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

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joule/internal/auth"
	"joule/internal/config"
	"joule/internal/db/sqlc"
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
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

func getUserID(r *http.Request) string {
	return r.Context().Value(auth.ContextUserID).(string)
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

	userID := getUserID(r)

	bmr, tdee, calorieTarget, proteinG, carbsG, fatG := CalculateTDEE(
		req.Sex, req.Age, req.WeightKg, req.HeightCm,
		req.ActivityLevel, req.Objective, req.DietPlan,
	)

	_, err := h.q.CreateProfile(r.Context(), sqlc.CreateProfileParams{
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
	userID := getUserID(r)

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

	userID := getUserID(r)

	err := h.q.UpdateProfile(r.Context(), sqlc.UpdateProfileParams{
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
		_, _, calorieTarget, proteinG, carbsG, fatG := CalculateTDEE(
			sex, age, weightKg, heightCm,
			activityLevel, goals.Objective, goals.DietPlan,
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
	userID := getUserID(r)

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
	userID := getUserID(r)

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

	userID := getUserID(r)
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

		_, _, cal, prot, carbs, fat := CalculateTDEE(sex, age, weightKg, heightCm, activityLevel, req.Objective, req.DietPlan)
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
	userID := getUserID(r)
	var prefs PreferencesResponse
	err := h.pool.QueryRow(r.Context(),
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
	userID := getUserID(r)
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
	_, err := h.pool.Exec(r.Context(),
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
