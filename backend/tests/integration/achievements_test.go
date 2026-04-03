package integration_test

import (
	"net/http"
	"testing"
)

func TestAchievementsRequiresAuth(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/achievements", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAchievementsWithAuth(t *testing.T) {
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
	_ = arr
}
