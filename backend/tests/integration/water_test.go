package integration_test

import (
	"net/http"
	"testing"
)

func TestWaterRequiresAuth(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/water", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestLogWater(t *testing.T) {
	token := getAdminToken(t)
	body := map[string]any{
		"amount_ml": 500,
	}
	resp := doRequest(t, http.MethodPost, "/api/water", body, token)
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}
