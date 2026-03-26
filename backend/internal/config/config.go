package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port         string
	AppURL       string
	DatabaseURL  string
	JWTSecret    string
	IsDev        bool
	AIProvider   string
	OpenAIKey    string
	AnthropicKey string
	AIModel      string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	UploadDir    string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "3000"),
		AppURL:      getEnv("APP_URL", "http://localhost:3000"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		AIProvider:  getEnv("AI_PROVIDER", "openai"),
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		AIModel:     os.Getenv("AI_MODEL"),
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPUser:    os.Getenv("SMTP_USER"),
		SMTPPass:    os.Getenv("SMTP_PASS"),
		UploadDir:   getEnv("UPLOAD_DIR", "./uploads"),
	}

	smtpPort := getEnv("SMTP_PORT", "587")
	port, err := strconv.Atoi(smtpPort)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}
	cfg.SMTPPort = port

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
