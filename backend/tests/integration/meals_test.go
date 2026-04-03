package integration_test

import (
	"net/http"
	"testing"
)

func TestGetMealsRequiresAuth(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/meals", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestGetMealsByDate(t *testing.T) {
	token := getAdminToken(t)
	resp := doRequest(t, http.MethodGet, "/api/meals?date=2026-04-03", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCreateMeal(t *testing.T) {
	token := getAdminToken(t)
	body := map[string]any{
		"meal_type": "lunch",
		"foods": []map[string]any{
			{
				"name":      "Test Food",
				"calories":  100,
				"protein_g": 5,
				"carbs_g":   10,
				"fat_g":     3,
			},
		},
	}
	resp := doRequest(t, http.MethodPost, "/api/meals", body, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		t.Fatalf("expected 200 or 201, got %d", resp.StatusCode)
	}
	data := readBody(t, resp)
	if _, ok := data["data"]; !ok {
		t.Fatalf("response missing data field")
	}
}

func TestDeleteMealRequiresAuth(t *testing.T) {
	resp := doRequest(t, http.MethodDelete, "/api/meals/some-id", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
