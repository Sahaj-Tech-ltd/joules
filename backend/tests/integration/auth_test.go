package integration_test

import (
	"net/http"
	"testing"
)

func TestLoginSuccess(t *testing.T) {
	body := map[string]string{
		"email":    "admin@joules.local",
		"password": "asdfghjk2003@P",
	}
	resp := doRequest(t, http.MethodPost, "/api/auth/login", body, "")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	data := readBody(t, resp)
	top, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatalf("response missing data object")
	}
	if _, ok := top["access_token"].(string); !ok {
		t.Fatalf("response missing access_token")
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	body := map[string]string{
		"email":    "admin@joules.local",
		"password": "wrongpassword",
	}
	resp := doRequest(t, http.MethodPost, "/api/auth/login", body, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestLoginMissingFields(t *testing.T) {
	body := map[string]string{}
	resp := doRequest(t, http.MethodPost, "/api/auth/login", body, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestMeRequiresAuth(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/auth/me", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestMeWithToken(t *testing.T) {
	token := getAdminToken(t)
	resp := doRequest(t, http.MethodGet, "/api/auth/me", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	data := readBody(t, resp)
	if _, ok := data["data"]; !ok {
		t.Fatalf("response missing data field with user info")
	}
}
