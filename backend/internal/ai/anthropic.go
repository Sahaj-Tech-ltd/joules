package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type AnthropicClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func newAnthropicClient(apiKey, model string) *AnthropicClient {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &AnthropicClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content []any  `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (c *AnthropicClient) Chat(systemPrompt string, messages []ChatMessage) (string, error) {
	var msgs []anthropicMessage
	for _, m := range messages {
		msgs = append(msgs, anthropicMessage{
			Role:    m.Role,
			Content: []any{m.Content},
		})
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: 1000,
		System:    systemPrompt,
		Messages:  msgs,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		slog.Error("anthropic chat error", "status", resp.StatusCode, "error", errBody)
		return "", fmt.Errorf("anthropic api returned status %d", resp.StatusCode)
	}

	var anthResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(anthResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return strings.TrimSpace(anthResp.Content[0].Text), nil
}

const anthropicSystemPrompt = `You are a nutrition analysis AI. Analyze the food in this image. Identify each food item visible. For each item, estimate calories, protein (g), carbs (g), fat (g), and fiber (g). Also estimate a reasonable serving size description. Return ONLY valid JSON as an array with no other text. Each item should have: name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, confidence (0-1). If no food is detected, return an empty array.`

func (c *AnthropicClient) IdentifyFood(imageData []byte) ([]IdentifiedFood, error) {
	mediaType := detectMediaType(imageData)

	b64 := base64.StdEncoding.EncodeToString(imageData)

	content := []any{
		map[string]any{
			"type": "image",
			"source": map[string]string{
				"type":       "base64",
				"media_type": mediaType,
				"data":       b64,
			},
		},
		map[string]any{
			"type": "text",
			"text": anthropicSystemPrompt,
		},
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: 1000,
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: content,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		slog.Error("anthropic api error", "status", resp.StatusCode, "error", errBody)
		return nil, fmt.Errorf("anthropic api returned status %d", resp.StatusCode)
	}

	var anthResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(anthResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	text := strings.TrimSpace(anthResp.Content[0].Text)
	text = strings.Trim(text, "```json")
	text = strings.Trim(text, "```")
	text = strings.TrimSpace(text)

	var foods []IdentifiedFood
	if err := json.Unmarshal([]byte(text), &foods); err != nil {
		return nil, fmt.Errorf("parse food json: %w", err)
	}

	return foods, nil
}
