package steps

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/auth"
	"joules/internal/config"
	"joules/internal/db/sqlc"
)

type Handler struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewHandler(q *sqlc.Queries, pool *pgxpool.Pool, cfg *config.Config) *Handler {
	return &Handler{q: q, pool: pool, cfg: cfg}
}

type apiResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type StepEntry struct {
	Date      string `json:"date"`
	StepCount int32  `json:"step_count"`
	Source    string `json:"source"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	slog.Error("request error", "status", status, "error", err)
	msg := err.Error()
	if status >= 500 {
		msg = "internal server error"
	}
	writeJSON(w, status, apiResponse{Error: msg})
}

func getUserID(r *http.Request) string {
	return r.Context().Value(auth.ContextUserID).(string)
}

func (h *Handler) GetSteps(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	dateStr := r.URL.Query().Get("date")

	var date time.Time
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date format: %w", err))
			return
		}
	} else {
		date = time.Now()
	}

	row, err := h.q.GetStepsByDate(r.Context(), sqlc.GetStepsByDateParams{
		UserID: userID,
		Date:   date,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusOK, apiResponse{Data: nil})
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get steps: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: StepEntry{
		Date:      row.Date.Format("2006-01-02"),
		StepCount: row.StepCount,
		Source:    row.Source,
	}})
}

func (h *Handler) LogSteps(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req struct {
		StepCount int32  `json:"step_count"`
		Date      string `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if req.StepCount < 0 {
		writeError(w, http.StatusBadRequest, errors.New("step_count must be non-negative"))
		return
	}

	var date time.Time
	if req.Date != "" {
		var err error
		date, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date format: %w", err))
			return
		}
	} else {
		date = time.Now()
	}

	row, err := h.q.LogSteps(r.Context(), sqlc.LogStepsParams{
		UserID:    userID,
		Date:      date,
		StepCount: req.StepCount,
		Source:    "manual",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("log steps: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: StepEntry{
		Date:      row.Date.Format("2006-01-02"),
		StepCount: row.StepCount,
		Source:    row.Source,
	}})
}

func (h *Handler) GetStepsHistory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from := time.Now().AddDate(0, 0, -6)
	to := time.Now()

	if fromStr != "" {
		t, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid from date: %w", err))
			return
		}
		from = t
	}
	if toStr != "" {
		t, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid to date: %w", err))
			return
		}
		to = t
	}

	rows, err := h.q.GetStepsHistory(r.Context(), sqlc.GetStepsHistoryParams{
		UserID: userID,
		Date:   from,
		Date_2: to,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get steps history: %w", err))
		return
	}

	entries := make([]StepEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, StepEntry{
			Date:      row.Date.Format("2006-01-02"),
			StepCount: row.StepCount,
			Source:    row.Source,
		})
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: entries})
}

func (h *Handler) GoogleStatus(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var exists bool
	err := h.pool.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM google_fit_tokens WHERE user_id = $1)",
		userID,
	).Scan(&exists)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("check google status: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"connected": exists}})
}

func (h *Handler) GoogleConnect(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	state := h.signState(userID)

	params := url.Values{}
	params.Set("client_id", h.cfg.GoogleClientID)
	params.Set("redirect_uri", h.cfg.AppURL+"/api/steps/google/callback")
	params.Set("response_type", "code")
	params.Set("scope", "https://www.googleapis.com/auth/fitness.activity.read")
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	params.Set("state", state)

	authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Redirect(w, r, "/settings?error=google_fit_missing_params", http.StatusFound)
		return
	}

	userID, err := h.verifyState(state)
	if err != nil {
		http.Redirect(w, r, "/settings?error=google_fit_invalid_state", http.StatusFound)
		return
	}

	token, err := h.exchangeCode(r.Context(), code)
	if err != nil {
		slog.Error("google fit token exchange failed", "error", err)
		http.Redirect(w, r, "/settings?error=google_fit_token_exchange", http.StatusFound)
		return
	}

	if err := h.saveToken(r.Context(), userID, token); err != nil {
		slog.Error("google fit save token failed", "error", err)
		http.Redirect(w, r, "/settings?error=google_fit_save_token", http.StatusFound)
		return
	}

	steps, err := h.fetchTodaySteps(r.Context(), token.AccessToken)
	if err != nil {
		slog.Warn("google fit initial sync failed (token saved)", "error", err)
	} else {
		today := time.Now()
		_, _ = h.q.LogSteps(r.Context(), sqlc.LogStepsParams{
			UserID:    userID,
			Date:      today,
			StepCount: int32(steps),
			Source:    "google_fit",
		})
	}

	http.Redirect(w, r, "/settings?connected=google_fit", http.StatusFound)
}

func (h *Handler) GoogleSync(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	ctx := r.Context()

	token, err := h.loadToken(ctx, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("google fit not connected"))
		return
	}

	if time.Now().After(token.Expiry) {
		token, err = h.refreshToken(ctx, userID, token.RefreshToken)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("refresh google token: %w", err))
			return
		}
	}

	steps, err := h.fetchTodaySteps(ctx, token.AccessToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("fetch google fit steps: %w", err))
		return
	}

	today := time.Now()
	row, err := h.q.LogSteps(ctx, sqlc.LogStepsParams{
		UserID:    userID,
		Date:      today,
		StepCount: int32(steps),
		Source:    "google_fit",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save synced steps: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]any{
		"synced_steps": row.StepCount,
		"date":         row.Date.Format("2006-01-02"),
	}})
}

func (h *Handler) signState(userID string) string {
	mac := hmac.New(sha256.New, []byte(h.cfg.GoogleClientSecret))
	mac.Write([]byte(userID))
	sig := hex.EncodeToString(mac.Sum(nil))
	return base64.URLEncoding.EncodeToString([]byte(userID + "." + sig))
}

func (h *Handler) verifyState(state string) (string, error) {
	data, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return "", fmt.Errorf("invalid state encoding")
	}
	parts := strings.SplitN(string(data), ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid state format")
	}
	userID := parts[0]
	mac := hmac.New(sha256.New, []byte(h.cfg.GoogleClientSecret))
	mac.Write([]byte(userID))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[1]), []byte(expected)) {
		return "", fmt.Errorf("invalid state signature")
	}
	return userID, nil
}

type oauthToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

func (h *Handler) exchangeCode(ctx context.Context, code string) (*oauthToken, error) {
	params := url.Values{}
	params.Set("code", code)
	params.Set("client_id", h.cfg.GoogleClientID)
	params.Set("client_secret", h.cfg.GoogleClientSecret)
	params.Set("redirect_uri", h.cfg.AppURL+"/api/steps/google/callback")
	params.Set("grant_type", "authorization_code")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", params)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("google oauth error: %s", result.Error)
	}

	return &oauthToken{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}

func (h *Handler) saveToken(ctx context.Context, userID string, token *oauthToken) error {
	_, err := h.pool.Exec(ctx,
		`INSERT INTO google_fit_tokens (user_id, access_token, refresh_token, expiry)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id) DO UPDATE
		     SET access_token = EXCLUDED.access_token,
		         refresh_token = CASE WHEN EXCLUDED.refresh_token != '' THEN EXCLUDED.refresh_token ELSE google_fit_tokens.refresh_token END,
		         expiry = EXCLUDED.expiry`,
		userID, token.AccessToken, token.RefreshToken, token.Expiry,
	)
	return err
}

func (h *Handler) loadToken(ctx context.Context, userID string) (*oauthToken, error) {
	var token oauthToken
	err := h.pool.QueryRow(ctx,
		"SELECT access_token, refresh_token, expiry FROM google_fit_tokens WHERE user_id = $1",
		userID,
	).Scan(&token.AccessToken, &token.RefreshToken, &token.Expiry)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (h *Handler) refreshToken(ctx context.Context, userID string, refreshToken string) (*oauthToken, error) {
	params := url.Values{}
	params.Set("refresh_token", refreshToken)
	params.Set("client_id", h.cfg.GoogleClientID)
	params.Set("client_secret", h.cfg.GoogleClientSecret)
	params.Set("grant_type", "refresh_token")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", params)
	if err != nil {
		return nil, fmt.Errorf("refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse refresh response: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("google refresh error: %s", result.Error)
	}

	token := &oauthToken{
		AccessToken:  result.AccessToken,
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}
	_ = h.saveToken(ctx, userID, token)
	return token, nil
}

func (h *Handler) fetchTodaySteps(ctx context.Context, accessToken string) (int64, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	body := fmt.Sprintf(`{
		"aggregateBy": [{"dataTypeName": "com.google.step_count.delta"}],
		"bucketByTime": {"durationMillis": %d},
		"startTimeMillis": %d,
		"endTimeMillis": %d
	}`, endOfDay.Sub(startOfDay).Milliseconds(), startOfDay.UnixMilli(), endOfDay.UnixMilli())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.googleapis.com/fitness/v1/users/me/dataset:aggregate",
		strings.NewReader(body),
	)
	if err != nil {
		return 0, fmt.Errorf("build fitness request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fitness api request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Bucket []struct {
			Dataset []struct {
				Point []struct {
					Value []struct {
						IntVal int64 `json:"intVal"`
					} `json:"value"`
				} `json:"point"`
			} `json:"dataset"`
		} `json:"bucket"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("parse fitness response: %w", err)
	}

	var total int64
	for _, bucket := range result.Bucket {
		for _, dataset := range bucket.Dataset {
			for _, point := range dataset.Point {
				for _, val := range point.Value {
					total += val.IntVal
				}
			}
		}
	}
	return total, nil
}
