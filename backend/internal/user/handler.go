package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

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

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	profile, err := h.q.GetProfile(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("profile not found: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: ProfileResponse{
		Name:               profile.Name,
		Age:                profile.Age,
		Sex:                profile.Sex,
		HeightCm:           numericToFloatPtr(profile.HeightCm),
		WeightKg:           numericToFloatPtr(profile.WeightKg),
		TargetWeightKg:     numericToFloatPtr(profile.TargetWeightKg),
		ActivityLevel:      profile.ActivityLevel,
		OnboardingComplete: profile.OnboardingComplete,
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

	writeJSON(w, http.StatusOK, apiResponse{Data: ProfileResponse{
		Name:               profile.Name,
		Age:                profile.Age,
		Sex:                profile.Sex,
		HeightCm:           numericToFloatPtr(profile.HeightCm),
		WeightKg:           numericToFloatPtr(profile.WeightKg),
		TargetWeightKg:     numericToFloatPtr(profile.TargetWeightKg),
		ActivityLevel:      profile.ActivityLevel,
		OnboardingComplete: profile.OnboardingComplete,
	}})
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
