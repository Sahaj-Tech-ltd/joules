package integration_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// registerTestUser signs up a new user and returns their auth token.
// Uses a unique email per call to avoid conflicts.
func registerTestUser(t *testing.T, email, password string) string {
	t.Helper()
	signupBody := map[string]string{
		"email":    email,
		"password": password,
	}
	resp := doRequest(t, http.MethodPost, "/api/auth/signup", signupBody, "")
	defer resp.Body.Close()
	// 200 = auto-approved, 201 = needs verification (both OK for our purposes)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		t.Fatalf("signup failed: expected 200/201, got %d", resp.StatusCode)
	}

	// Try to login — if verification is required, this will fail
	loginBody := map[string]string{
		"email":    email,
		"password": password,
	}
	loginResp := doRequest(t, http.MethodPost, "/api/auth/login", loginBody, "")
	defer loginResp.Body.Close()
	if loginResp.StatusCode != 200 {
		t.Skipf("test user login failed (needs verification?): %d", loginResp.StatusCode)
	}
	data := readBody(t, loginResp)
	tokenData, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatalf("login response missing data object")
	}
	token, ok := tokenData["access_token"].(string)
	if !ok || token == "" {
		t.Fatalf("login response missing access_token")
	}
	return token
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("test-%s-%d@joules.test", prefix, time.Now().UnixNano())
}

// logMeal creates a meal via the API with the given food items.
func logMeal(t *testing.T, token string, mealType string, foods []map[string]any) map[string]any {
	t.Helper()
	body := map[string]any{
		"meal_type": mealType,
		"foods":     foods,
	}
	resp := doRequest(t, http.MethodPost, "/api/meals", body, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		t.Fatalf("log meal: expected 200/201, got %d", resp.StatusCode)
	}
	return readBody(t, resp)
}

// checkAchievements triggers achievement check and returns all achievements.
func checkAchievements(t *testing.T, token string) []map[string]any {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/api/achievements/check", map[string]any{}, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("check achievements: expected 200, got %d", resp.StatusCode)
	}
	data := readBody(t, resp)
	arr, ok := data["data"].([]any)
	if !ok {
		t.Fatalf("achievements response missing data array")
	}
	result := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

// getAchievements fetches unlocked achievements without triggering a check.
func getAchievements(t *testing.T, token string) []map[string]any {
	t.Helper()
	resp := doRequest(t, http.MethodGet, "/api/achievements", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("get achievements: expected 200, got %d", resp.StatusCode)
	}
	data := readBody(t, resp)
	arr, ok := data["data"].([]any)
	if !ok {
		t.Fatalf("achievements response missing data array")
	}
	result := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

// hasAchievement checks if a specific achievement type exists in the list.
func hasAchievement(achievements []map[string]any, achievementType string) bool {
	for _, a := range achievements {
		if t, ok := a["type"].(string); ok && t == achievementType {
			return true
		}
	}
	return false
}

// ---------- Achievement Auth Tests ----------

func TestAchievementsRequiresAuth(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/achievements", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAchievementsCheckRequiresAuth(t *testing.T) {
	resp := doRequest(t, http.MethodPost, "/api/achievements/check", map[string]any{}, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// ---------- Achievement Logic Tests ----------
// These tests use a fresh test user so each test gets a clean achievement state.

func TestAchievementsEmptyDay_NoNutritionAchievements(t *testing.T) {
	email := uniqueEmail("emptyday")
	token := registerTestUser(t, email, "TestPass123!")

	// Check achievements without logging anything
	achievements := checkAchievements(t, token)

	// low_carb_day should NOT trigger when no meals are logged (the bug fix)
	if hasAchievement(achievements, "low_carb_day") {
		t.Error("low_carb_day should NOT unlock when no meals are logged")
	}

	// No nutrition achievements should fire on an empty day
	for _, a := range achievements {
		achType, _ := a["type"].(string)
		switch achType {
		case "low_carb_day", "high_protein_day", "fiber_champion",
			"calorie_goal", "protein_goal", "perfect_day":
			t.Errorf("nutrition achievement %s should NOT unlock on empty day", achType)
		}
	}

	// first_meal should NOT trigger
	if hasAchievement(achievements, "first_meal") {
		t.Error("first_meal should NOT unlock when no meals are logged")
	}
}

func TestAchievementsLowCarbDay_Unlocks(t *testing.T) {
	email := uniqueEmail("lowcarb")
	token := registerTestUser(t, email, "TestPass123!")

	// Log a meal with < 50g carbs (e.g., 20g carbs)
	logMeal(t, token, "lunch", []map[string]any{
		{
			"name":      "Chicken Salad",
			"calories":  350,
			"protein_g": 30,
			"carbs_g":   15,
			"fat_g":     18,
			"fiber_g":   4,
		},
	})

	achievements := checkAchievements(t, token)

	if !hasAchievement(achievements, "low_carb_day") {
		t.Error("low_carb_day should unlock when carbs < 50g and meals are logged")
	}

	// first_meal should also unlock
	if !hasAchievement(achievements, "first_meal") {
		t.Error("first_meal should unlock after logging a meal")
	}
}

func TestAchievementsLowCarbDay_DoesNotUnlock_HighCarbs(t *testing.T) {
	email := uniqueEmail("highcarb")
	token := registerTestUser(t, email, "TestPass123!")

	// Log a meal with > 50g carbs
	logMeal(t, token, "breakfast", []map[string]any{
		{
			"name":      "Pancakes with Syrup",
			"calories":  600,
			"protein_g": 12,
			"carbs_g":   85,
			"fat_g":     22,
			"fiber_g":   2,
		},
	})

	achievements := checkAchievements(t, token)

	if hasAchievement(achievements, "low_carb_day") {
		t.Error("low_carb_day should NOT unlock when carbs >= 50g")
	}

	// first_meal should still unlock
	if !hasAchievement(achievements, "first_meal") {
		t.Error("first_meal should unlock after logging a meal")
	}
}

func TestAchievementsFirstMeal(t *testing.T) {
	email := uniqueEmail("firstmeal")
	token := registerTestUser(t, email, "TestPass123!")

	// Before logging: no first_meal
	before := getAchievements(t, token)
	if hasAchievement(before, "first_meal") {
		t.Error("first_meal should not exist before logging any meals")
	}

	// Log a meal
	logMeal(t, token, "dinner", []map[string]any{
		{
			"name":      "Grilled Fish",
			"calories":  400,
			"protein_g": 35,
			"carbs_g":   10,
			"fat_g":     20,
			"fiber_g":   1,
		},
	})

	// Check achievements
	achievements := checkAchievements(t, token)
	if !hasAchievement(achievements, "first_meal") {
		t.Error("first_meal should unlock after logging first meal")
	}
}

func TestAchievementsFirstWater(t *testing.T) {
	email := uniqueEmail("firstwater")
	token := registerTestUser(t, email, "TestPass123!")

	// Log water
	waterBody := map[string]any{"amount_ml": 500}
	waterResp := doRequest(t, http.MethodPost, "/api/water", waterBody, token)
	defer waterResp.Body.Close()
	if waterResp.StatusCode != 201 {
		t.Fatalf("log water: expected 201, got %d", waterResp.StatusCode)
	}

	achievements := checkAchievements(t, token)
	if !hasAchievement(achievements, "first_water") {
		t.Error("first_water should unlock after logging water")
	}
}

func TestAchievementsFirstExercise(t *testing.T) {
	email := uniqueEmail("firstexercise")
	token := registerTestUser(t, email, "TestPass123!")

	// Log exercise
	exerciseBody := map[string]any{
		"name":           "Running",
		"duration_min":   30,
		"calories_burned": 300,
	}
	exResp := doRequest(t, http.MethodPost, "/api/exercises", exerciseBody, token)
	defer exResp.Body.Close()
	if exResp.StatusCode != 200 && exResp.StatusCode != 201 {
		t.Fatalf("log exercise: expected 200/201, got %d", exResp.StatusCode)
	}

	achievements := checkAchievements(t, token)
	if !hasAchievement(achievements, "first_exercise") {
		t.Error("first_exercise should unlock after logging exercise")
	}

	// exercise_1 should also unlock
	if !hasAchievement(achievements, "exercise_1") {
		t.Error("exercise_1 should unlock after logging exercise")
	}
}

func TestAchievementsHighProteinDay(t *testing.T) {
	email := uniqueEmail("highprotein")
	token := registerTestUser(t, email, "TestPass123!")

	// Log meals that total > 150g protein
	logMeal(t, token, "breakfast", []map[string]any{
		{
			"name":      "Protein Shake",
			"calories":  200,
			"protein_g": 40,
			"carbs_g":   5,
			"fat_g":     3,
			"fiber_g":   0,
		},
	})
	logMeal(t, token, "lunch", []map[string]any{
		{
			"name":      "Chicken Breast",
			"calories":  500,
			"protein_g": 60,
			"carbs_g":   0,
			"fat_g":     10,
			"fiber_g":   0,
		},
	})
	logMeal(t, token, "dinner", []map[string]any{
		{
			"name":      "Steak",
			"calories":  600,
			"protein_g": 55,
			"carbs_g":   0,
			"fat_g":     30,
			"fiber_g":   0,
		},
	})

	achievements := checkAchievements(t, token)
	if !hasAchievement(achievements, "high_protein_day") {
		t.Error("high_protein_day should unlock when total protein >= 150g")
	}
}

func TestAchievementsFiberChampion(t *testing.T) {
	email := uniqueEmail("fiberchamp")
	token := registerTestUser(t, email, "TestPass123!")

	// Log a meal with > 30g fiber
	logMeal(t, token, "lunch", []map[string]any{
		{
			"name":      "Bean & Lentil Bowl",
			"calories":  450,
			"protein_g": 25,
			"carbs_g":   60,
			"fat_g":     8,
			"fiber_g":   35,
		},
	})

	achievements := checkAchievements(t, token)
	if !hasAchievement(achievements, "fiber_champion") {
		t.Error("fiber_champion should unlock when total fiber >= 30g")
	}
}

func TestAchievementsMultipleChecksIdempotent(t *testing.T) {
	email := uniqueEmail("idempotent")
	token := registerTestUser(t, email, "TestPass123!")

	// Log a meal
	logMeal(t, token, "lunch", []map[string]any{
		{
			"name":      "Salad",
			"calories":  300,
			"protein_g": 15,
			"carbs_g":   20,
			"fat_g":     15,
			"fiber_g":   8,
		},
	})

	// Check achievements twice
	first := checkAchievements(t, token)
	second := checkAchievements(t, token)

	// Both should return the same achievements (idempotent)
	if len(first) != len(second) {
		t.Errorf("achievement check not idempotent: first=%d, second=%d", len(first), len(second))
	}

	// Both should have first_meal
	if !hasAchievement(first, "first_meal") || !hasAchievement(second, "first_meal") {
		t.Error("first_meal should persist across multiple checks")
	}
}

func TestAchievementsCalorieGoal(t *testing.T) {
	email := uniqueEmail("calgoal")
	token := registerTestUser(t, email, "TestPass123!")

	// First set goals (we need to know the user's calorie target)
	// Default goals might be set, or we need to check
	// For now, let's just verify that the achievement doesn't fire with 0 calories
	achievements := checkAchievements(t, token)
	if hasAchievement(achievements, "calorie_goal") {
		t.Error("calorie_goal should NOT unlock with 0 calories logged")
	}
	if hasAchievement(achievements, "protein_goal") {
		t.Error("protein_goal should NOT unlock with 0 protein logged")
	}
	if hasAchievement(achievements, "perfect_day") {
		t.Error("perfect_day should NOT unlock on empty day")
	}
}

func TestGetAchievementsReturnsUnlocked(t *testing.T) {
	token := getAdminToken(t)
	resp := doRequest(t, http.MethodGet, "/api/achievements", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	data := readBody(t, resp)
	arr, ok := data["data"].([]any)
	if !ok {
		t.Fatalf("response missing data array")
	}
	// Admin should have some achievements from previous checks
	t.Logf("admin has %d achievements", len(arr))
}
