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

type OpenAIClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func newOpenAIClient(apiKey, model string) *OpenAIClient {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

type openaiChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type openaiChatRequest struct {
	Model    string             `json:"model"`
	MaxTokens int               `json:"max_tokens"`
	Messages []openaiChatMessage `json:"messages"`
}

type openaiChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *OpenAIClient) Chat(systemPrompt string, messages []ChatMessage) (string, error) {
	msgs := []openaiChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	for _, m := range messages {
		msgs = append(msgs, openaiChatMessage{Role: m.Role, Content: m.Content})
	}

	reqBody := openaiChatRequest{
		Model:     c.model,
		MaxTokens: 1000,
		Messages:  msgs,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		slog.Error("openai chat error", "status", resp.StatusCode, "error", errBody)
		return "", fmt.Errorf("openai api returned status %d", resp.StatusCode)
	}

	var chatResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

const openaiSystemPrompt = `You are a nutrition analysis AI. Analyze the food in this image. Identify each food item visible. For each item, estimate calories, protein (g), carbs (g), fat (g), and fiber (g). Also estimate a reasonable serving size description. Return ONLY valid JSON as an array with no other text. Each item should have: name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, confidence (0-1). If no food is detected, return an empty array.`

func (c *OpenAIClient) IdentifyFood(imageData []byte) ([]IdentifiedFood, error) {
	mediaType := detectMediaType(imageData)

	b64 := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, b64)

	userContent := []any{
		map[string]any{
			"type": "image_url",
			"image_url": map[string]string{
				"url": dataURL,
			},
		},
	}

	reqBody := openaiChatRequest{
		Model:     c.model,
		MaxTokens: 1000,
		Messages: []openaiChatMessage{
			{Role: "system", Content: openaiSystemPrompt},
			{Role: "user", Content: userContent},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		slog.Error("openai api error", "status", resp.StatusCode, "error", errBody)
		return nil, fmt.Errorf("openai api returned status %d", resp.StatusCode)
	}

	var chatResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	content = strings.Trim(content, "```json")
	content = strings.Trim(content, "```")
	content = strings.TrimSpace(content)

	var foods []IdentifiedFood
	if err := json.Unmarshal([]byte(content), &foods); err != nil {
		return nil, fmt.Errorf("parse food json: %w", err)
	}

	return foods, nil
}
