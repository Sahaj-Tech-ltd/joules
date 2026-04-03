package integration_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestExportCSVRequiresAuth(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/export/csv", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestExportCSVWithAuth(t *testing.T) {
	token := getAdminToken(t)
	resp := doRequest(t, http.MethodGet, "/api/export/csv", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected content-type to contain text/csv, got %s", ct)
	}
}

func TestExportJSONWithAuth(t *testing.T) {
	token := getAdminToken(t)
	resp := doRequest(t, http.MethodGet, "/api/export/json", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "json") {
		t.Fatalf("expected content-type to contain json, got %s", ct)
	}
}

func TestExportCSVWeight(t *testing.T) {
	token := getAdminToken(t)
	resp := doRequest(t, http.MethodGet, "/api/export/csv?type=weight", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExportCSVWater(t *testing.T) {
	token := getAdminToken(t)
	resp := doRequest(t, http.MethodGet, "/api/export/csv?type=water", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExportJSONWeight(t *testing.T) {
	token := getAdminToken(t)
	resp := doRequest(t, http.MethodGet, "/api/export/json?type=weight", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
