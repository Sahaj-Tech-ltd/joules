package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port               string
	AppURL             string
	DatabaseURL        string
	JWTSecret          string
	IsDev              bool
	AIProvider         string
	OpenAIKey          string
	OpenAIBaseURL      string
	AnthropicKey       string
	AIModel            string
	VisionModel        string
	OCRModel           string
	RoutingModel       string
	ClassifierModel    string
	TavilyAPIKey       string
	OCRProvider        string
	VAPIDPublicKey     string
	VAPIDPrivateKey    string
	VAPIDContact       string
	NtfyBaseURL        string
	NtfyToken          string
	SMTPHost           string
	SMTPPort           int
	SMTPUser           string
	SMTPPass           string
	UploadDir          string
	AdminEmail         string
	RequireApproval    bool
	GoogleClientID     string
	GoogleClientSecret string

	VisionAPIKey      string
	VisionBaseURL     string
	OCRAPIKey         string
	OCRBaseURL        string
	ClassifierAPIKey  string
	ClassifierBaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "3000"),
		AppURL:             getEnv("APP_URL", "http://localhost:3000"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		AIProvider:         getEnv("AI_PROVIDER", "openai"),
		OpenAIKey:          os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL:      getEnv("OPENAI_BASE_URL", "https://api.openai.com"),
		AnthropicKey:       os.Getenv("ANTHROPIC_API_KEY"),
		AIModel:            os.Getenv("AI_MODEL"),
		VisionModel:        os.Getenv("VISION_MODEL"),
		OCRModel:           os.Getenv("OCR_MODEL"),
		RoutingModel:       os.Getenv("ROUTING_MODEL"),
		ClassifierModel:    os.Getenv("CLASSIFIER_MODEL"),
		TavilyAPIKey:       os.Getenv("TAVILY_API_KEY"),
		OCRProvider:        getEnv("OCR_PROVIDER", ""),
		VAPIDPublicKey:     os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:    os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDContact:       getEnv("VAPID_CONTACT", "mailto:admin@example.com"),
		NtfyBaseURL:        os.Getenv("NTFY_BASE_URL"),
		NtfyToken:          os.Getenv("NTFY_TOKEN"),
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPUser:           os.Getenv("SMTP_USER"),
		SMTPPass:           os.Getenv("SMTP_PASS"),
		UploadDir:          getEnv("UPLOAD_DIR", "./uploads"),
		AdminEmail:         getEnv("ADMIN_EMAIL", "admin@joules.local"),
		RequireApproval:    os.Getenv("REQUIRE_APPROVAL") == "true",
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),

		VisionAPIKey:      os.Getenv("VISION_API_KEY"),
		VisionBaseURL:     os.Getenv("VISION_BASE_URL"),
		OCRAPIKey:         os.Getenv("OCR_API_KEY"),
		OCRBaseURL:        os.Getenv("OCR_BASE_URL"),
		ClassifierAPIKey:  os.Getenv("CLASSIFIER_API_KEY"),
		ClassifierBaseURL: os.Getenv("CLASSIFIER_BASE_URL"),
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
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
