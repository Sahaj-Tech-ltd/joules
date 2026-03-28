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
	Provider      string
	OpenAIKey     string
	OpenAIBaseURL string
	AnthropicKey  string
	Model         string
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
	switch cfg.Provider {
	case "openai":
		if cfg.OpenAIKey == "" {
			return nil, nil
		}
		return newOpenAIClient(cfg.OpenAIKey, cfg.Model, cfg.OpenAIBaseURL), nil
	case "anthropic":
		if cfg.AnthropicKey == "" {
			return nil, nil
		}
		return newAnthropicClient(cfg.AnthropicKey, cfg.Model), nil
	default:
		return nil, nil
	}
}
