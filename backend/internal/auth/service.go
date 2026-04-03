package auth

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"joules/internal/config"
	"joules/internal/db/sqlc"
	syslog "joules/internal/syslog"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserExists         = errors.New("email already registered")
	ErrNotVerified        = errors.New("account not verified")
	ErrInvalidCode        = errors.New("invalid verification code")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrNotApproved        = errors.New("account pending admin approval")
)

type Service struct {
	q    *sqlc.Queries
	cfg  *config.Config
	pool *pgxpool.Pool
}

func NewService(q *sqlc.Queries, pool *pgxpool.Pool, cfg *config.Config) *Service {
	return &Service{q: q, cfg: cfg, pool: pool}
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
	var hasUpper, hasNumber, hasSymbol bool
	const symbols = "!@#$%^&*()_+-=[]{}|;':\",./<>?"
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasNumber = true
		default:
			if strings.ContainsRune(symbols, c) {
				hasSymbol = true
			}
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
	if !hasSymbol {
		return errors.New("password must contain at least one symbol (!@#$% etc.)")
	}
	return nil
}

func (s *Service) generateTokenPair(userID string) (*TokenResponse, error) {
	now := time.Now()
	accessExpiry := now.Add(24 * time.Hour)
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

	existing, err := s.q.GetUserByEmail(context.Background(), email)
	if err == nil && existing.Email != "" {
		return nil, ErrUserExists
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	code := generateCode()

	user, err := s.q.CreateUser(context.Background(), sqlc.CreateUserParams{
		Email:            email,
		PasswordHash:     hash,
		VerificationCode: &code,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	smtpHost, _, _, _ := s.effectiveSMTP()
	if smtpHost != "" {
		if err := s.sendVerificationEmail(email, code); err != nil {
			slog.Warn("failed to send verification email", "error", err, "email", email)
			slog.Info("verification code (email send failed)", "email", email, "code", code)
			syslog.Error("smtp", "Failed to send verification email", map[string]any{"email": email, "error": err.Error(), "date": time.Now().Format("2006-01-02")})
		} else {
			slog.Info("verification email sent", "email", email)
			syslog.Info("smtp", "Verification email sent", map[string]any{"email": email, "date": time.Now().Format("2006-01-02")})
		}
	} else {
		slog.Info("verification code (no SMTP configured)", "user_id", user.ID, "email", email, "code", code)
		syslog.Info("smtp", "SMTP not configured — verification code logged only", map[string]any{"email": email})
	}

	if s.cfg.RequireApproval {
		s.pool.Exec(context.Background(), "UPDATE users SET approved = FALSE WHERE id = $1", user.ID)
	}

	return &SignupResponse{
		Message: "Account created. Please verify your email.",
	}, nil
}

// effectiveSMTP returns SMTP settings from app_settings (DB overrides) falling back to env config.
// This allows admins to update SMTP via the admin panel without a restart.
func (s *Service) effectiveSMTP() (host string, port int, user, pass string) {
	host, port, user, pass = s.cfg.SMTPHost, s.cfg.SMTPPort, s.cfg.SMTPUser, s.cfg.SMTPPass

	rows, err := s.pool.Query(context.Background(),
		"SELECT key, value FROM app_settings WHERE key IN ('smtp_host', 'smtp_port', 'smtp_user', 'smtp_pass')")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		switch k {
		case "smtp_host":
			if v != "" {
				host = v
			}
		case "smtp_port":
			if p, err2 := strconv.Atoi(v); err2 == nil && p > 0 {
				port = p
			}
		case "smtp_user":
			if v != "" {
				user = v
			}
		case "smtp_pass":
			if v != "" {
				pass = v
			}
		}
	}
	return
}

// loginAuth implements the SMTP LOGIN auth mechanism (used by cPanel/Exim on port 465).
type loginAuth struct{ user, pass string }

func (a loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}
func (a loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch strings.ToLower(string(fromServer)) {
		case "username:":
			return []byte(a.user), nil
		case "password:":
			return []byte(a.pass), nil
		}
	}
	return nil, nil
}

func (s *Service) sendVerificationEmail(email, code string) error {
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>Your Joule verification code</title></head>
<body style="margin:0;padding:0;background-color:#f1f5f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f1f5f9;padding:40px 16px;">
    <tr><td align="center">
      <table width="100%%" cellpadding="0" cellspacing="0" style="max-width:600px;background-color:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,0.08);">
        <!-- Header -->
        <tr>
          <td style="background-color:#0f172a;padding:32px 40px;text-align:center;">
            <span style="font-size:32px;font-weight:800;color:#f59e0b;letter-spacing:-0.5px;">&#9889; Joule</span>
          </td>
        </tr>
        <!-- Body -->
        <tr>
          <td style="padding:40px 40px 32px;text-align:center;">
            <p style="margin:0 0 8px;font-size:22px;font-weight:700;color:#0f172a;">Welcome to Joule!</p>
            <p style="margin:0 0 32px;font-size:15px;color:#64748b;line-height:1.6;">Here&rsquo;s your verification code to complete your signup:</p>
            <!-- Code box -->
            <div style="display:inline-block;background-color:#f8fafc;border:2px solid #f59e0b;border-radius:12px;padding:20px 40px;margin-bottom:32px;">
              <span style="font-family:'Courier New',Courier,monospace;font-size:36px;font-weight:700;color:#0f172a;letter-spacing:8px;">%s</span>
            </div>
            <p style="margin:0;font-size:13px;color:#94a3b8;line-height:1.6;">Enter this code in the app to activate your account.<br>This code does not expire.</p>
          </td>
        </tr>
        <!-- Footer -->
        <tr>
          <td style="background-color:#f8fafc;padding:20px 40px;text-align:center;border-top:1px solid #e2e8f0;">
            <p style="margin:0;font-size:12px;color:#94a3b8;">If you didn&rsquo;t sign up for Joule, you can safely ignore this email.</p>
          </td>
        </tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, code)

	plainBody := fmt.Sprintf("Hi,\r\n\r\nYour Joule verification code is:\r\n\r\n  %s\r\n\r\nEnter this in the app to complete your signup. It won't expire.\r\n\r\nIf you didn't sign up for Joule, ignore this email.\r\n", code)

	smtpHost, smtpPort, smtpUser, smtpPass := s.effectiveSMTP()

	boundary := "joule-boundary-28f7a3b1"
	msg := fmt.Sprintf(
		"From: Joule <%s>\r\nTo: %s\r\nSubject: Your Joule verification code: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n--%s\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s\r\n--%s\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n--%s--\r\n",
		smtpUser, email, code, boundary,
		boundary, plainBody,
		boundary, htmlBody,
		boundary,
	)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	// Port 465 uses implicit TLS (SMTPS); port 587 uses STARTTLS
	if smtpPort == 465 {
		tlsConn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: smtpHost})
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(tlsConn, smtpHost)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Auth(loginAuth{smtpUser, smtpPass}); err != nil {
			return err
		}
		if err := c.Mail(smtpUser); err != nil {
			return err
		}
		if err := c.Rcpt(email); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, msg); err != nil {
			return err
		}
		return w.Close()
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	return smtp.SendMail(addr, auth, smtpUser, []string{email}, []byte(msg))
}

func (s *Service) Verify(email, code string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.q.GetUserByEmail(context.Background(), email)
	if err != nil {
		return ErrInvalidCode
	}

	if user.Verified {
		return nil
	}

	if user.VerificationCode == nil || *user.VerificationCode != code {
		return ErrInvalidCode
	}

	return s.q.VerifyUser(context.Background(), sqlc.VerifyUserParams{
		ID:               user.ID,
		VerificationCode: &code,
	})
}

func (s *Service) Login(email, password string) (*LoginResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.q.GetUserByEmail(context.Background(), email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !checkPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	if !user.Verified {
		return nil, ErrNotVerified
	}

	var approved bool
	if err := s.pool.QueryRow(context.Background(), "SELECT approved FROM users WHERE id = $1", user.ID).Scan(&approved); err == nil && !approved {
		return nil, ErrNotApproved
	}

	tokens, err := s.generateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	var mustChange bool
	s.pool.QueryRow(context.Background(), "SELECT must_change_password FROM users WHERE id = $1", user.ID).Scan(&mustChange)

	return &LoginResponse{
		AccessToken:        tokens.AccessToken,
		RefreshToken:       tokens.RefreshToken,
		ExpiresAt:          tokens.ExpiresAt,
		MustChangePassword: mustChange,
	}, nil
}

// ChangePassword updates a user's password, verifying the old one first.
func (s *Service) ChangePassword(userID, oldPassword, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	var hash string
	if err := s.pool.QueryRow(context.Background(), "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&hash); err != nil {
		return ErrInvalidCredentials
	}
	var mustChange bool
	_ = s.pool.QueryRow(context.Background(), "SELECT must_change_password FROM users WHERE id = $1", userID).Scan(&mustChange)
	if !mustChange {
		if !checkPassword(hash, oldPassword) {
			return ErrInvalidCredentials
		}
	}
	newHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(context.Background(),
		"UPDATE users SET password_hash = $1, must_change_password = FALSE WHERE id = $2",
		newHash, userID,
	)
	return err
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

// EnsureAdminUser seeds the admin account on first run.
// Returns a one-time setup token if the account was just created, "" if admin already exists.
func (s *Service) EnsureAdminUser() string {
	if s.cfg.AdminEmail == "" {
		return ""
	}
	ctx := context.Background()
	var count int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE is_admin = TRUE").Scan(&count); err != nil || count > 0 {
		return ""
	}
	email := strings.TrimSpace(strings.ToLower(s.cfg.AdminEmail))
	// Placeholder hash — never used; real password set via setup flow
	placeholder, err := hashPassword(generateTempPassword())
	if err != nil {
		slog.Error("failed to create admin placeholder hash", "error", err)
		return ""
	}
	token := generateSetupToken()
	_, err = s.pool.Exec(ctx,
		`INSERT INTO users (email, password_hash, verified, approved, is_admin, must_change_password, verification_code)
		 VALUES ($1, $2, TRUE, TRUE, TRUE, TRUE, $3)
		 ON CONFLICT (email) DO UPDATE
		   SET is_admin = TRUE, verified = TRUE, approved = TRUE,
		       must_change_password = TRUE, verification_code = $3`,
		email, placeholder, token,
	)
	if err != nil {
		slog.Error("failed to create admin user", "error", err)
		return ""
	}
	return token
}

func generateSetupToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b) // 48-char hex string
}

func generateTempPassword() string {
	const chars = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789!@#$"
	buf := make([]byte, 16)
	for i := range buf {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		buf[i] = chars[n.Int64()]
	}
	return string(buf)
}

// CompleteSetup validates a first-run setup token and returns a JWT for the admin.
// The token is consumed (cleared) on success.
func (s *Service) CompleteSetup(token string) (*TokenResponse, error) {
	ctx := context.Background()
	var userID string
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM users WHERE verification_code = $1 AND is_admin = TRUE AND must_change_password = TRUE`,
		token,
	).Scan(&userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE users SET verification_code = NULL WHERE id = $1`, userID,
	)
	if err != nil {
		return nil, err
	}
	return s.generateTokenPair(userID)
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
