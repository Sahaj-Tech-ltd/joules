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
	apiKey          string
	model           string
	visionModel     string
	ocrModel        string
	classifierModel string
	httpClient      *http.Client
	prompts         map[string]string
}

func newAnthropicClient(apiKey, model, visionModel, ocrModel, classifierModel string, prompts map[string]string) *AnthropicClient {
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &AnthropicClient{
		apiKey:          apiKey,
		model:           model,
		visionModel:     visionModel,
		ocrModel:        ocrModel,
		classifierModel: classifierModel,
		httpClient:      &http.Client{},
		prompts:         prompts,
	}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content []any  `json:"content"`
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type anthropicResponse struct {
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
}

func (c *AnthropicClient) Chat(systemPrompt string, messages []ChatMessage) (string, error) {
	var msgs []anthropicMessage
	for _, m := range messages {
		msgs = append(msgs, anthropicMessage{
			Role:    m.Role,
			Content: []any{map[string]string{"type": "text", "text": m.Content}},
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

	// Find the first text block
	for _, block := range anthResp.Content {
		if block.Type == "text" {
			return strings.TrimSpace(block.Text), nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

func (c *AnthropicClient) ChatAgent(systemPrompt string, messages []ChatMessage, tools []Tool) (*AgentResponse, error) {
	if len(tools) == 0 {
		text, err := c.Chat(systemPrompt, messages)
		if err != nil {
			return nil, err
		}
		return &AgentResponse{Content: text}, nil
	}

	// Convert tools to Anthropic format
	anthTools := make([]anthropicTool, 0, len(tools))
	for _, t := range tools {
		anthTools = append(anthTools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	// Convert messages to Anthropic format
	var msgs []anthropicMessage
	for _, m := range messages {
		switch m.Role {
		case "assistant":
			// Rebuild assistant message with any tool_use blocks
			var contentBlocks []any
			if m.Content != "" {
				contentBlocks = append(contentBlocks, map[string]string{
					"type": "text",
					"text": m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				// tc.Args is a JSON string; unmarshal to raw for Anthropic
				var inputRaw json.RawMessage
				if tc.Args != "" {
					inputRaw = json.RawMessage(tc.Args)
				} else {
					inputRaw = json.RawMessage("{}")
				}
				contentBlocks = append(contentBlocks, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": inputRaw,
				})
			}
			msgs = append(msgs, anthropicMessage{Role: "assistant", Content: contentBlocks})

		case "tool":
			// Tool result — must be a user message with tool_result block
			msgs = append(msgs, anthropicMessage{
				Role: "user",
				Content: []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": m.ToolCallID,
						"content":     m.Content,
					},
				},
			})

		default:
			// user message
			msgs = append(msgs, anthropicMessage{
				Role: m.Role,
				Content: []any{
					map[string]string{"type": "text", "text": m.Content},
				},
			})
		}
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: 2000,
		System:    systemPrompt,
		Messages:  msgs,
		Tools:     anthTools,
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
		slog.Error("anthropic agent error", "status", resp.StatusCode, "error", errBody)
		return nil, fmt.Errorf("anthropic api returned status %d", resp.StatusCode)
	}

	var anthResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	agentResp := &AgentResponse{}

	for _, block := range anthResp.Content {
		switch block.Type {
		case "text":
			agentResp.Content = strings.TrimSpace(block.Text)
		case "tool_use":
			var argsStr string
			if block.Input != nil {
				argsStr = string(block.Input)
			} else {
				argsStr = "{}"
			}
			agentResp.ToolCalls = append(agentResp.ToolCalls, ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: argsStr,
			})
		}
	}

	return agentResp, nil
}

const anthropicSystemPrompt = `You are a nutrition analysis assistant. Your only job is to identify food in images and return structured nutrition data.

Instructions:
- Identify every distinct food or drink item visible in the image.
- OCR priority: If the image contains any text — nutrition labels, ingredient lists, menu items, restaurant receipts, product packaging, barcode labels — READ that text first and use it as the ground truth for nutrition values. Text data is always more accurate than visual estimation.
- For packaged items: read the Nutrition Facts panel if visible. Use the exact values for calories, protein, carbs, fat, and fiber from the label.
- For menus or receipts: read the dish names and use those exact names for identification.
- Estimate portion size using visual cues: plate diameter, hand size, packaging volume, context clues. If the user provides a portion description, use it as the primary reference.
- For restaurant or takeaway food, assume a standard restaurant serving unless told otherwise.
- For homemade food, estimate conservatively.
- Return ONLY a raw JSON array — no markdown, no code fences, no explanation text.
- Each element: { "name": string, "calories": number, "protein_g": number, "carbs_g": number, "fat_g": number, "fiber_g": number, "serving_size": string, "confidence": number (0-1) }
- confidence: 0.95+ for values read directly from a nutrition label, 0.6-0.8 for estimated portions, below 0.5 for unclear items.
- If no food is visible, return [].`

func (c *AnthropicClient) IdentifyFood(imageData []byte, hint string) ([]IdentifiedFood, error) {
	mediaType := detectMediaType(imageData)

	b64 := base64.StdEncoding.EncodeToString(imageData)

	visionPrompt := anthropicSystemPrompt
	if c.prompts["prompt_vision"] != "" {
		visionPrompt = c.prompts["prompt_vision"]
	}

	textContent := visionPrompt
	if hint != "" {
		textContent += "\n\nPortion/context from user: " + hint
	}

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
			"text": textContent,
		},
	}

	reqBody := anthropicRequest{
		Model:     c.visionModel,
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

	// Find the first text block
	var text string
	for _, block := range anthResp.Content {
		if block.Type == "text" {
			text = strings.TrimSpace(block.Text)
			break
		}
	}
	if text == "" {
		return nil, fmt.Errorf("no text content in response")
	}

	if strings.HasPrefix(text, "```") {
		if idx := strings.Index(text, "\n"); idx != -1 {
			text = text[idx+1:]
		}
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	var foods []IdentifiedFood
	if err := json.Unmarshal([]byte(text), &foods); err != nil {
		return nil, fmt.Errorf("parse food json: %w", err)
	}

	return foods, nil
}

const anthropicOCRParsePrompt = `You are a nutrition analysis assistant. Extract food items and their nutrition values from the provided OCR text.

Instructions:
- The text was extracted via OCR from a food photo (nutrition label, menu, receipt, packaging, or food description).
- Parse every distinct food or drink item mentioned.
- For Nutrition Facts labels: use the exact calorie, protein, carbs, fat, and fiber values from the label.
- For menus or food descriptions: estimate macros from standard nutritional databases.
- Return ONLY a raw JSON array — no markdown, no code fences, no explanation.
- Each element: { "name": string, "calories": number, "protein_g": number, "carbs_g": number, "fat_g": number, "fiber_g": number, "serving_size": string, "confidence": number (0-1) }
- confidence: 0.95+ for values read from a label, 0.6-0.8 for estimates.
- If no food data is found, return [].`

func (c *AnthropicClient) IdentifyFoodFromText(ocrText, hint string) ([]IdentifiedFood, error) {
	userContent := "OCR text from food image:\n\n" + ocrText
	if hint != "" {
		userContent += "\n\nUser context: " + hint
	}

	ocrPrompt := anthropicOCRParsePrompt
	if c.prompts["prompt_ocr"] != "" {
		ocrPrompt = c.prompts["prompt_ocr"]
	}

	m := c.ocrModel
	reqBody := anthropicRequest{
		Model:     m,
		MaxTokens: 1000,
		System:    ocrPrompt,
		Messages:  []anthropicMessage{{Role: "user", Content: []any{map[string]string{"type": "text", "text": userContent}}}},
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
		return nil, fmt.Errorf("anthropic api returned status %d", resp.StatusCode)
	}

	var anthResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	for _, block := range anthResp.Content {
		if block.Type == "text" {
			content = strings.TrimSpace(block.Text)
			break
		}
	}

	if strings.HasPrefix(content, "```") {
		if idx := strings.Index(content, "\n"); idx != -1 {
			content = content[idx+1:]
		}
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	var foods []IdentifiedFood
	if err := json.Unmarshal([]byte(content), &foods); err != nil {
		return nil, fmt.Errorf("parse food json: %w", err)
	}
	return foods, nil
}

const anthropicClassifierPrompt = `Classify this image into exactly one category. Reply with only one word:
- "food_photo" if the image shows prepared food, meals, or drinks
- "receipt" if the image shows a restaurant receipt, bill, or order summary
- "nutrition_label" if the image shows a nutrition facts panel, ingredient list, or product packaging with text

Reply with only the category name, nothing else.`

const anthropicTextExtractionPrompt = `Extract all visible text from this image. This is an image of a food receipt, nutrition label, or packaging. 
Return ALL the text you can see, preserving the structure as closely as possible. 
Do not summarize or interpret — just transcribe the text exactly as it appears.
If there is no text visible, reply with NONE.`

func (c *AnthropicClient) ClassifyImage(imageData []byte) (string, error) {
	mediaType := detectMediaType(imageData)
	b64 := base64.StdEncoding.EncodeToString(imageData)

	classifierPrompt := anthropicClassifierPrompt
	if c.prompts["prompt_classifier"] != "" {
		classifierPrompt = c.prompts["prompt_classifier"]
	}

	content := []any{
		map[string]any{
			"type": "image",
			"source": map[string]string{
				"type":       "base64",
				"media_type": mediaType,
				"data":       b64,
			},
		},
	}

	reqBody := anthropicRequest{
		Model:     c.classifierModel,
		MaxTokens: 10,
		System:    classifierPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: content},
		},
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
		slog.Error("anthropic classify error", "status", resp.StatusCode, "error", errBody)
		return "", fmt.Errorf("anthropic api returned status %d", resp.StatusCode)
	}

	var anthResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	for _, block := range anthResp.Content {
		if block.Type == "text" {
			return strings.ToLower(strings.TrimSpace(block.Text)), nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

func (c *AnthropicClient) ExtractTextFromImage(imageData []byte) (string, error) {
	mediaType := detectMediaType(imageData)
	b64 := base64.StdEncoding.EncodeToString(imageData)

	textExtractPrompt := anthropicTextExtractionPrompt
	if c.prompts["prompt_text_extract"] != "" {
		textExtractPrompt = c.prompts["prompt_text_extract"]
	}

	content := []any{
		map[string]any{
			"type": "image",
			"source": map[string]string{
				"type":       "base64",
				"media_type": mediaType,
				"data":       b64,
			},
		},
	}

	reqBody := anthropicRequest{
		Model:     c.ocrModel,
		MaxTokens: 2000,
		System:    textExtractPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: content},
		},
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
		slog.Error("anthropic text extraction error", "status", resp.StatusCode, "error", errBody)
		return "", fmt.Errorf("anthropic api returned status %d", resp.StatusCode)
	}

	var anthResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	for _, block := range anthResp.Content {
		if block.Type == "text" {
			text := strings.TrimSpace(block.Text)
			if text == "NONE" {
				return "", nil
			}
			return text, nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}
