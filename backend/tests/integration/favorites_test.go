package integration_test

import (
	"net/http"
	"testing"
)

func TestFavoritesCRUD(t *testing.T) {
	token := getAdminToken(t)

	createBody := map[string]any{
		"name":      "Test Fav",
		"calories":  200,
		"protein_g": 10,
		"carbs_g":   20,
		"fat_g":     5,
	}
	resp := doRequest(t, http.MethodPost, "/api/favorites", createBody, token)
	defer resp.Body.Close()
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		t.Fatalf("create favorite: expected 201 or 200, got %d", resp.StatusCode)
	}
	data := readBody(t, resp)
	favData, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatalf("create favorite response missing data object")
	}
	favID, ok := favData["id"].(string)
	if !ok || favID == "" {
		t.Fatalf("create favorite response missing id")
	}

	resp = doRequest(t, http.MethodGet, "/api/favorites", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("list favorites: expected 200, got %d", resp.StatusCode)
	}
	listData := readBody(t, resp)
	items, ok := listData["data"].([]any)
	if !ok {
		t.Fatalf("list favorites response missing data array")
	}
	found := false
	for _, item := range items {
		fav, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if fav["id"] == favID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created favorite %s not found in list", favID)
	}

	resp = doRequest(t, http.MethodGet, "/api/favorites/top", nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("top favorites: expected 200, got %d", resp.StatusCode)
	}

	resp = doRequest(t, http.MethodDelete, "/api/favorites/"+favID, nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("delete favorite: expected 200, got %d", resp.StatusCode)
	}
}

func TestFavoritesRequireAuth(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/favorites", nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
