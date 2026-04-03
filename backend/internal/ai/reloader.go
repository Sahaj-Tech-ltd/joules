package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReloadingClient struct {
	pool     *pgxpool.Pool
	envCfg   Config
	mu       sync.RWMutex
	current  Client
	lastLoad time.Time
	ttl      time.Duration
}

func NewReloadingClient(pool *pgxpool.Pool, envCfg Config) *ReloadingClient {
	rc := &ReloadingClient{
		pool:   pool,
		envCfg: envCfg,
		ttl:    30 * time.Second,
	}
	// Initial load from env config (before DB is available or has settings)
	client, _ := NewClient(envCfg)
	rc.current = client
	return rc
}

func (rc *ReloadingClient) getClient() Client {
	rc.mu.RLock()
	if rc.current != nil && time.Since(rc.lastLoad) < rc.ttl {
		client := rc.current
		rc.mu.RUnlock()
		return client
	}
	rc.mu.RUnlock()

	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Double-check after acquiring write lock
	if rc.current != nil && time.Since(rc.lastLoad) < rc.ttl {
		return rc.current
	}

	client := rc.loadFromDB()
	rc.current = client
	rc.lastLoad = time.Now()
	return rc.current
}

func (rc *ReloadingClient) loadFromDB() Client {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	getSetting := func(key string) string {
		var val string
		rc.pool.QueryRow(ctx, "SELECT value FROM app_settings WHERE key = $1", key).Scan(&val)
		return val
	}

	provider := getSetting("ai_provider")
	if provider == "" {
		provider = rc.envCfg.Provider
	}

	model := getSetting("ai_model")
	if model == "" {
		model = rc.envCfg.Model
	}

	visionModel := getSetting("vision_model")
	if visionModel == "" {
		visionModel = rc.envCfg.VisionModel
	}
	if visionModel == "" {
		visionModel = model
	}

	ocrModel := getSetting("ocr_model")
	if ocrModel == "" {
		ocrModel = getSetting("routing_model") // backward compat
	}
	if ocrModel == "" {
		ocrModel = rc.envCfg.OCRModel
	}
	if ocrModel == "" {
		ocrModel = rc.envCfg.RoutingModel
	}
	if ocrModel == "" {
		ocrModel = model
	}

	openaiKey := rc.envCfg.OpenAIKey
	anthropicKey := rc.envCfg.AnthropicKey
	baseURL := rc.envCfg.OpenAIBaseURL

	switch provider {
	case "custom":
		if customURL := getSetting("custom_base_url"); customURL != "" {
			baseURL = customURL
		}
		if customKey := getSetting("custom_api_key"); customKey != "" {
			openaiKey = customKey
		}
		provider = "openai" // custom providers use OpenAI-compatible API
	}

	prompts := make(map[string]string)
	promptRows, err := rc.pool.Query(ctx, "SELECT key, value FROM app_settings WHERE key LIKE 'prompt_%'")
	if err == nil {
		for promptRows.Next() {
			var k, v string
			if promptRows.Scan(&k, &v) == nil {
				prompts[k] = v
			}
		}
		promptRows.Close()
	}
	if len(prompts) == 0 {
		prompts = rc.envCfg.Prompts
	}

	cfg := Config{
		Provider:      provider,
		OpenAIKey:     openaiKey,
		OpenAIBaseURL: baseURL,
		AnthropicKey:  anthropicKey,
		Model:         model,
		VisionModel:   visionModel,
		OCRModel:      ocrModel,
		Prompts:       prompts,
	}

	client, err := NewClient(cfg)
	if err != nil {
		slog.Error("failed to create AI client from DB config", "error", err)
		return nil
	}
	slog.Info("AI client reloaded from DB", "provider", provider, "model", model, "vision", visionModel, "ocr", ocrModel)
	return client
}

// ForceReload forces a reload on next call (useful after admin saves settings)
func (rc *ReloadingClient) ForceReload() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.lastLoad = time.Time{} // zero time = always expired
}

// Implement Client interface by delegating to current client

func (rc *ReloadingClient) IdentifyFood(imageData []byte, hint string) ([]IdentifiedFood, error) {
	client := rc.getClient()
	if client == nil {
		return nil, fmt.Errorf("AI client not configured")
	}
	return client.IdentifyFood(imageData, hint)
}

func (rc *ReloadingClient) IdentifyFoodFromText(ocrText, hint string) ([]IdentifiedFood, error) {
	client := rc.getClient()
	if client == nil {
		return nil, fmt.Errorf("AI client not configured")
	}
	return client.IdentifyFoodFromText(ocrText, hint)
}

func (rc *ReloadingClient) Chat(systemPrompt string, messages []ChatMessage) (string, error) {
	client := rc.getClient()
	if client == nil {
		return "", fmt.Errorf("AI client not configured")
	}
	return client.Chat(systemPrompt, messages)
}

func (rc *ReloadingClient) ChatAgent(systemPrompt string, messages []ChatMessage, tools []Tool) (*AgentResponse, error) {
	client := rc.getClient()
	if client == nil {
		return nil, fmt.Errorf("AI client not configured")
	}
	return client.ChatAgent(systemPrompt, messages, tools)
}

func (rc *ReloadingClient) ClassifyImage(imageData []byte) (string, error) {
	client := rc.getClient()
	if client == nil {
		return "", fmt.Errorf("AI client not configured")
	}
	return client.ClassifyImage(imageData)
}

func (rc *ReloadingClient) ExtractTextFromImage(imageData []byte) (string, error) {
	client := rc.getClient()
	if client == nil {
		return "", fmt.Errorf("AI client not configured")
	}
	return client.ExtractTextFromImage(imageData)
}
