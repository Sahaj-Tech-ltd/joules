package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/smtp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"joule/internal/config"
	"joule/internal/db/sqlc"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserExists         = errors.New("email already registered")
	ErrNotVerified        = errors.New("account not verified")
	ErrInvalidCode        = errors.New("invalid verification code")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type Service struct {
	q   *sqlc.Queries
	cfg *config.Config
}

func NewService(q *sqlc.Queries, cfg *config.Config) *Service {
	return &Service{q: q, cfg: cfg}
}

func generateCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func validatePassword(password string) error {
	var hasUpper, hasNumber bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasNumber = true
		}
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	return nil
}

func (s *Service) generateTokenPair(userID string) (*TokenResponse, error) {
	now := time.Now()
	accessExpiry := now.Add(15 * time.Minute)
	refreshExpiry := now.Add(7 * 24 * time.Hour)

	accessClaims := jwt.MapClaims{
		"sub": userID,
		"exp": accessExpiry.Unix(),
		"iat": now.Unix(),
		"typ": "access",
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	refreshClaims := jwt.MapClaims{
		"sub": userID,
		"exp": refreshExpiry.Unix(),
		"iat": now.Unix(),
		"typ": "refresh",
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresAt:    accessExpiry,
	}, nil
}

func (s *Service) Signup(email, password string) (*SignupResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if err := validatePassword(password); err != nil {
		return nil, err
	}

	existing, err := s.q.GetUserByEmail(nil, email)
	if err == nil && existing.Email != "" {
		return nil, ErrUserExists
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	code := generateCode()

	user, err := s.q.CreateUser(nil, sqlc.CreateUserParams{
		Email:            email,
		PasswordHash:     hash,
		VerificationCode: &code,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	if s.cfg.SMTPHost != "" {
		if err := s.sendVerificationEmail(email, code); err != nil {
			slog.Warn("failed to send verification email", "error", err, "email", email)
			slog.Info("verification code (email send failed)", "email", email, "code", code)
		} else {
			slog.Info("verification email sent", "email", email)
		}
	} else {
		slog.Info("verification code (no SMTP configured)", "user_id", user.ID, "email", email, "code", code)
	}

	return &SignupResponse{
		Message: "Account created. Please verify your email.",
	}, nil
}

func (s *Service) sendVerificationEmail(email, code string) error {
	subject := "Subject: Joule - Verify Your Email\n"
	body := fmt.Sprintf(
		"\r\nYour verification code is: %s\r\n\r\nEnter this code in the app to complete your signup.\r\n",
		code,
	)
	msg := subject + "MIME-version: 1.0;\nContent-type: text/plain; charset=\"UTF-8\";\n\n" + body

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	return smtp.SendMail(addr, auth, s.cfg.SMTPUser, []string{email}, []byte(msg))
}

func (s *Service) Verify(email, code string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.q.GetUserByEmail(nil, email)
	if err != nil {
		return ErrInvalidCode
	}

	if user.Verified {
		return nil
	}

	if user.VerificationCode == nil || *user.VerificationCode != code {
		return ErrInvalidCode
	}

	return s.q.VerifyUser(nil, sqlc.VerifyUserParams{
		ID:               user.ID,
		VerificationCode: &code,
	})
}

func (s *Service) Login(email, password string) (*LoginResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.q.GetUserByEmail(nil, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !checkPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	if !user.Verified {
		return nil, ErrNotVerified
	}

	tokens, err := s.generateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt,
	}, nil
}

func (s *Service) RefreshToken(refreshToken string) (*TokenResponse, error) {
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	typ, _ := claims["typ"].(string)
	if typ != "refresh" {
		return nil, ErrInvalidToken
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, ErrInvalidToken
	}

	return s.generateTokenPair(sub)
}

func ParseToken(secret, tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	typ, _ := claims["typ"].(string)
	if typ != "access" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
