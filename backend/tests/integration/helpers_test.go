package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const baseURL = "http://localhost:3000"

func getAdminToken(t *testing.T) string {
	t.Helper()
	body := map[string]string{
		"email":    "admin@joules.local",
		"password": "asdfghjk2003@P",
	}
	resp := doRequest(t, http.MethodPost, "/api/auth/login", body, "")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("login failed: expected 200, got %d", resp.StatusCode)
	}
	var result map[string]any
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading login body: %v", err)
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("parsing login body: %v", err)
	}
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("login response missing data object")
	}
	token, ok := data["access_token"].(string)
	if !ok || token == "" {
		t.Fatalf("login response missing access_token")
	}
	return token
}

func doRequest(t *testing.T, method, path string, body any, token string) *http.Response {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshaling request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("making request: %v", err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("parsing response body: %v", err)
	}
	return result
}
