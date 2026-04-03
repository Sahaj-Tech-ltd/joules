package ai

type IdentifiedFood struct {
	Name        string  `json:"name"`
	Calories    float64 `json:"calories"`
	ProteinG    float64 `json:"protein_g"`
	CarbsG      float64 `json:"carbs_g"`
	FatG        float64 `json:"fat_g"`
	FiberG      float64 `json:"fiber_g"`
	ServingSize string  `json:"serving_size"`
	Confidence  float64 `json:"confidence"`
}

type Config struct {
	Provider        string
	OpenAIKey       string
	OpenAIBaseURL   string
	AnthropicKey    string
	Model           string
	VisionModel     string
	OCRModel        string
	RoutingModel    string
	ClassifierModel string
	Prompts         map[string]string
}

func (c Config) VisionPrompt() string {
	if p, ok := c.Prompts["prompt_vision"]; ok && p != "" {
		return p
	}
	return ""
}

func (c Config) OCRPrompt() string {
	if p, ok := c.Prompts["prompt_ocr"]; ok && p != "" {
		return p
	}
	return ""
}

// Tool represents a callable tool the AI can use.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{} // JSON Schema
}

// ToolCall represents a tool invocation requested by the AI.
type ToolCall struct {
	ID   string
	Name string
	Args string // JSON string of arguments
}

// AgentResponse is the result of a ChatAgent call.
type AgentResponse struct {
	Content   string
	ToolCalls []ToolCall
}

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // for role="tool" messages
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // for role="assistant" with tool calls
}

type Client interface {
	// hint is an optional user-provided description of portion size / context.
	// Pass "" if not provided.
	IdentifyFood(imageData []byte, hint string) ([]IdentifiedFood, error)
	// IdentifyFoodFromText parses nutrition data from OCR-extracted text.
	// Used when OCR_PROVIDER=tesseract — avoids sending the image to the vision model.
	IdentifyFoodFromText(ocrText, hint string) ([]IdentifiedFood, error)
	ClassifyImage(imageData []byte) (string, error)
	ExtractTextFromImage(imageData []byte) (string, error)
	Chat(systemPrompt string, messages []ChatMessage) (string, error)
	// ChatAgent runs a chat with tool-calling support.
	// If tools is empty or the provider doesn't support tools, falls back to Chat.
	ChatAgent(systemPrompt string, messages []ChatMessage, tools []Tool) (*AgentResponse, error)
}

func detectMediaType(data []byte) string {
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	if len(data) >= 4 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 {
		return "image/webp"
	}
	return "image/jpeg"
}

func NewClient(cfg Config) (Client, error) {
	visionModel := cfg.VisionModel
	if visionModel == "" {
		visionModel = cfg.Model
	}
	ocrModel := cfg.OCRModel
	if ocrModel == "" {
		ocrModel = cfg.RoutingModel
	}
	if ocrModel == "" {
		ocrModel = cfg.Model
	}
	classifierModel := cfg.ClassifierModel
	if classifierModel == "" {
		classifierModel = ocrModel
	}
	if classifierModel == "" {
		classifierModel = cfg.Model
	}

	switch cfg.Provider {
	case "openai":
		if cfg.OpenAIKey == "" {
			return nil, nil
		}
		return newOpenAIClient(cfg.OpenAIKey, cfg.Model, visionModel, ocrModel, classifierModel, cfg.OpenAIBaseURL, cfg.Prompts), nil
	case "anthropic":
		if cfg.AnthropicKey == "" {
			return nil, nil
		}
		return newAnthropicClient(cfg.AnthropicKey, cfg.Model, visionModel, ocrModel, classifierModel, cfg.Prompts), nil
	default:
		return nil, nil
	}
}
