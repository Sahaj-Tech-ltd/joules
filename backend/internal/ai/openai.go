package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type OpenAIClient struct {
	apiKey          string
	model           string
	visionModel     string
	ocrModel        string
	classifierModel string
	baseURL         string
	httpClient      *http.Client
	prompts         map[string]string
}

func newOpenAIClient(apiKey, model, visionModel, ocrModel, classifierModel, baseURL string, prompts map[string]string) *OpenAIClient {
	if model == "" {
		model = "gpt-5.4-mini-2026-03-17"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &OpenAIClient{
		apiKey:          apiKey,
		model:           model,
		visionModel:     visionModel,
		ocrModel:        ocrModel,
		classifierModel: classifierModel,
		baseURL:         baseURL,
		httpClient:      &http.Client{Timeout: 60 * time.Second},
		prompts:         prompts,
	}
}

func (c *OpenAIClient) useMaxCompletion() bool {
	return strings.Contains(c.baseURL, "openai.com")
}

type openaiChatMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
}

type openaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openaiTool struct {
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openaiChatRequest struct {
	Model            string              `json:"model"`
	MaxTokens        int                 `json:"-"`
	Messages         []openaiChatMessage `json:"messages"`
	Tools            []openaiTool        `json:"tools,omitempty"`
	UseMaxCompletion bool                `json:"-"`
}

func (r openaiChatRequest) MarshalJSON() ([]byte, error) {
	type Alias openaiChatRequest
	a := struct {
		Alias
		MaxCompTok *int `json:"max_completion_tokens,omitempty"`
		MaxTok     *int `json:"max_tokens,omitempty"`
	}{Alias: Alias(r)}
	if r.MaxTokens > 0 {
		if r.UseMaxCompletion {
			a.MaxCompTok = &r.MaxTokens
		} else {
			a.MaxTok = &r.MaxTokens
		}
	}
	return json.Marshal(a)
}

type openaiChatResponse struct {
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openaiToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
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
		Model:            c.model,
		MaxTokens:        1000,
		Messages:         msgs,
		UseMaxCompletion: c.useMaxCompletion(),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
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

func (c *OpenAIClient) ChatAgent(systemPrompt string, messages []ChatMessage, tools []Tool) (*AgentResponse, error) {
	if len(tools) == 0 {
		text, err := c.Chat(systemPrompt, messages)
		if err != nil {
			return nil, err
		}
		return &AgentResponse{Content: text}, nil
	}

	msgs := []openaiChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	for _, m := range messages {
		msg := openaiChatMessage{
			Role:       m.Role,
			ToolCallID: m.ToolCallID,
		}
		// For assistant messages with tool calls, content may be empty
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			// Reconstruct openai tool calls
			oaiToolCalls := make([]openaiToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				otc := openaiToolCall{
					ID:   tc.ID,
					Type: "function",
				}
				otc.Function.Name = tc.Name
				otc.Function.Arguments = tc.Args
				oaiToolCalls = append(oaiToolCalls, otc)
			}
			msg.ToolCalls = oaiToolCalls
			msg.Content = m.Content // may be empty string
		} else {
			msg.Content = m.Content
		}
		msgs = append(msgs, msg)
	}

	// Convert tools to OpenAI format
	oaiTools := make([]openaiTool, 0, len(tools))
	for _, t := range tools {
		oaiTools = append(oaiTools, openaiTool{
			Type: "function",
			Function: openaiToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	reqBody := openaiChatRequest{
		Model:            c.model,
		MaxTokens:        2000,
		Messages:         msgs,
		Tools:            oaiTools,
		UseMaxCompletion: c.useMaxCompletion(),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	slog.Info("openai ChatAgent request", "model", c.model, "messages", len(msgs), "tools", len(oaiTools), "baseURL", c.baseURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		slog.Error("openai agent error", "status", resp.StatusCode, "error", errBody, "model", c.model)
		return nil, fmt.Errorf("openai api returned status %d: %v", resp.StatusCode, errBody)
	}

	var chatResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := chatResp.Choices[0]
	agentResp := &AgentResponse{
		Content: strings.TrimSpace(choice.Message.Content),
	}

	for _, tc := range choice.Message.ToolCalls {
		agentResp.ToolCalls = append(agentResp.ToolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: tc.Function.Arguments,
		})
	}

	return agentResp, nil
}

const openaiSystemPrompt = `You are a nutrition analysis assistant. Your only job is to identify food in images and return structured nutrition data.

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

func (c *OpenAIClient) IdentifyFood(imageData []byte, hint string) ([]IdentifiedFood, error) {
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
	if hint != "" {
		userContent = append(userContent, map[string]any{
			"type": "text",
			"text": "Portion/context from user: " + hint,
		})
	}

	visionPrompt := openaiSystemPrompt
	if c.prompts["prompt_vision"] != "" {
		visionPrompt = c.prompts["prompt_vision"]
	}

	reqBody := openaiChatRequest{
		Model:     c.visionModel,
		MaxTokens: 1000,
		Messages: []openaiChatMessage{
			{Role: "system", Content: visionPrompt},
			{Role: "user", Content: userContent},
		},
		UseMaxCompletion: c.useMaxCompletion(),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
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
	// Strip markdown code fences if the model wraps output despite instructions
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

const openaiOCRParsePrompt = `You are a nutrition analysis assistant. Extract food items and their nutrition values from the provided OCR text.

Instructions:
- The text was extracted via OCR from a food photo (nutrition label, menu, receipt, packaging, or food description).
- Parse every distinct food or drink item mentioned.
- For Nutrition Facts labels: use the exact calorie, protein, carbs, fat, and fiber values from the label.
- For menus or food descriptions: estimate macros from standard nutritional databases.
- Return ONLY a raw JSON array — no markdown, no code fences, no explanation.
- Each element: { "name": string, "calories": number, "protein_g": number, "carbs_g": number, "fat_g": number, "fiber_g": number, "serving_size": string, "confidence": number (0-1) }
- confidence: 0.95+ for values read from a label, 0.6-0.8 for estimates.
- If no food data is found, return [].`

func (c *OpenAIClient) IdentifyFoodFromText(ocrText, hint string) ([]IdentifiedFood, error) {
	userContent := "OCR text from food image:\n\n" + ocrText
	if hint != "" {
		userContent += "\n\nUser context: " + hint
	}

	ocrPrompt := openaiOCRParsePrompt
	if c.prompts["prompt_ocr"] != "" {
		ocrPrompt = c.prompts["prompt_ocr"]
	}

	m := c.ocrModel
	reqBody := openaiChatRequest{
		Model:     m,
		MaxTokens: 1000,
		Messages: []openaiChatMessage{
			{Role: "system", Content: ocrPrompt},
			{Role: "user", Content: userContent},
		},
		UseMaxCompletion: c.useMaxCompletion(),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
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

const openaiClassifierPrompt = `Classify this image into exactly one category. Reply with only one word:
- "food_photo" if the image shows prepared food, meals, or drinks
- "receipt" if the image shows a restaurant receipt, bill, or order summary
- "nutrition_label" if the image shows a nutrition facts panel, ingredient list, or product packaging with text

Reply with only the category name, nothing else.`

const openaiTextExtractionPrompt = `Extract all visible text from this image. This is an image of a food receipt, nutrition label, or packaging. 
Return ALL the text you can see, preserving the structure as closely as possible. 
Do not summarize or interpret — just transcribe the text exactly as it appears.
If there is no text visible, reply with NONE.`

func (c *OpenAIClient) ClassifyImage(imageData []byte) (string, error) {
	mediaType := detectMediaType(imageData)
	b64 := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, b64)

	classifierPrompt := openaiClassifierPrompt
	if c.prompts["prompt_classifier"] != "" {
		classifierPrompt = c.prompts["prompt_classifier"]
	}

	reqBody := openaiChatRequest{
		Model:            c.classifierModel,
		MaxTokens:        10,
		UseMaxCompletion: c.useMaxCompletion(),
		Messages: []openaiChatMessage{
			{Role: "system", Content: classifierPrompt},
			{Role: "user", Content: []any{
				map[string]any{
					"type": "image_url",
					"image_url": map[string]string{
						"url": dataURL,
					},
				},
			}},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
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
		slog.Error("openai classify error", "status", resp.StatusCode, "error", errBody)
		return "", fmt.Errorf("openai api returned status %d", resp.StatusCode)
	}

	var chatResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.ToLower(strings.TrimSpace(chatResp.Choices[0].Message.Content)), nil
}

func (c *OpenAIClient) ExtractTextFromImage(imageData []byte) (string, error) {
	mediaType := detectMediaType(imageData)
	b64 := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, b64)

	textExtractPrompt := openaiTextExtractionPrompt
	if c.prompts["prompt_text_extract"] != "" {
		textExtractPrompt = c.prompts["prompt_text_extract"]
	}

	reqBody := openaiChatRequest{
		Model:            c.ocrModel,
		MaxTokens:        2000,
		UseMaxCompletion: c.useMaxCompletion(),
		Messages: []openaiChatMessage{
			{Role: "system", Content: textExtractPrompt},
			{Role: "user", Content: []any{
				map[string]any{
					"type": "image_url",
					"image_url": map[string]string{
						"url": dataURL,
					},
				},
			}},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
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
		slog.Error("openai text extraction error", "status", resp.StatusCode, "error", errBody)
		return "", fmt.Errorf("openai api returned status %d", resp.StatusCode)
	}

	var chatResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	text := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if text == "NONE" {
		return "", nil
	}
	return text, nil
}
