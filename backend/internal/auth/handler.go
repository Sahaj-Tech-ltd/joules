package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
	msg := err.Error()
	if status >= 500 {
		msg = "internal server error"
	}
	writeJSON(w, status, apiResponse{Error: msg})
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	resp, err := h.svc.Signup(req.Email, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case ErrUserExists:
			status = http.StatusConflict
		case ErrInvalidCredentials:
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: resp})
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.Verify(req.Email, req.Code); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "Email verified successfully"}})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	resp, err := h.svc.Login(req.Email, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		switch err {
		case ErrNotVerified:
			status = http.StatusForbidden
		case ErrNotApproved:
			status = http.StatusForbidden
		}
		writeError(w, status, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   !h.svc.cfg.IsDev,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	writeJSON(w, http.StatusOK, apiResponse{Data: LoginResponse{
		AccessToken:        resp.AccessToken,
		ExpiresAt:          resp.ExpiresAt,
		MustChangePassword: resp.MustChangePassword,
	}})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		cookie, err2 := r.Cookie("refresh_token")
		if err2 != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		req.RefreshToken = cookie.Value
	}

	resp, err := h.svc.RefreshToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   !h.svc.cfg.IsDev,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextUserID).(string)
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"id": userID}})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ContextUserID).(string)
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, errors.New("new_password is required"))
		return
	}
	if err := h.svc.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		status := http.StatusInternalServerError
		if err == ErrInvalidCredentials {
			status = http.StatusUnauthorized
		} else if err.Error() == "password must be at least 8 characters" {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "password changed"}})
}

type setupCompleteRequest struct {
	Token string `json:"token"`
}

func (h *Handler) SetupComplete(w http.ResponseWriter, r *http.Request) {
	var req setupCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		writeError(w, http.StatusBadRequest, errors.New("token required"))
		return
	}
	resp, err := h.svc.CompleteSetup(req.Token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, errors.New("invalid or expired setup token"))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

type contextKey string

// ContextUserID is the exported context key for the authenticated user's ID.
const ContextUserID contextKey = "userID"

func AdminMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, _ := r.Context().Value(ContextUserID).(string)
			if userID == "" {
				writeError(w, http.StatusForbidden, errors.New("admin access required"))
				return
			}
			var isAdmin bool
			if err := pool.QueryRow(r.Context(), "SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin); err != nil || !isAdmin {
				writeError(w, http.StatusForbidden, errors.New("admin access required"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, ErrInvalidToken)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				writeError(w, http.StatusUnauthorized, ErrInvalidToken)
				return
			}

			claims, err := ParseToken(secret, tokenStr)
			if err != nil {
				writeError(w, http.StatusUnauthorized, err)
				return
			}

			sub, _ := claims["sub"].(string)
			if sub == "" {
				writeError(w, http.StatusUnauthorized, ErrInvalidToken)
				return
			}

			ctx := context.WithValue(r.Context(), ContextUserID, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
