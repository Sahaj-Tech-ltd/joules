package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type tavilyRequest struct {
	APIKey          string `json:"api_key"`
	Query           string `json:"query"`
	SearchDepth     string `json:"search_depth"`
	MaxResults      int    `json:"max_results"`
	IncludeAnswer   bool   `json:"include_answer"`
}

type tavilyResponse struct {
	Answer  string `json:"answer"`
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

// SearchWeb calls the Tavily search API and returns the top results as formatted text.
func SearchWeb(apiKey, query string) (string, error) {
	reqBody := tavilyRequest{
		APIKey:        apiKey,
		Query:         query,
		SearchDepth:   "basic",
		MaxResults:    3,
		IncludeAnswer: true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal tavily request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create tavily request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tavily request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("tavily api returned status %d: %v", resp.StatusCode, errBody)
	}

	var tavilyResp tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tavilyResp); err != nil {
		return "", fmt.Errorf("decode tavily response: %w", err)
	}

	var sb strings.Builder

	if tavilyResp.Answer != "" {
		sb.WriteString("Summary: ")
		sb.WriteString(tavilyResp.Answer)
		sb.WriteString("\n\n")
	}

	for i, result := range tavilyResp.Results {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, result.Title))
		sb.WriteString(result.URL)
		sb.WriteString("\n")
		if result.Content != "" {
			// Truncate long content
			content := result.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			sb.WriteString(content)
		}
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String()), nil
}
