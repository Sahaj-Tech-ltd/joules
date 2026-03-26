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
	Provider     string
	OpenAIKey    string
	AnthropicKey string
	Model        string
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client interface {
	IdentifyFood(imageData []byte) ([]IdentifiedFood, error)
	Chat(systemPrompt string, messages []ChatMessage) (string, error)
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
		return newOpenAIClient(cfg.OpenAIKey, cfg.Model), nil
	case "anthropic":
		if cfg.AnthropicKey == "" {
			return nil, nil
		}
		return newAnthropicClient(cfg.AnthropicKey, cfg.Model), nil
	default:
		return nil, nil
	}
}
