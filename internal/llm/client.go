package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const summarizeSystemPrompt = `You are a Kubernetes log analyst. Analyze the following logs and provide:
1. A brief summary of what the service is doing
2. Any errors or warnings found
3. Any anomalies or things worth investigating`

const questionSystemPrompt = `You are a Kubernetes log analyst. Analyze the following logs to answer the user's question.`

// Client sends logs to an OpenAI-compatible LLM endpoint.
type Client struct {
	baseURL string
	apiKey  string
	model   string
}

func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, model: model}
}

// Ask sends logs to the LLM. If question is empty, summarization mode is used.
func (c *Client) Ask(logs, question string) (string, error) {
	systemPrompt := summarizeSystemPrompt
	userContent := logs
	if question != "" {
		systemPrompt = questionSystemPrompt
		userContent = fmt.Sprintf("Logs:\n%s\n\nQuestion: %s", logs, question)
	}

	body, err := json.Marshal(map[string]any{
		"model": c.model,
		"messages": []map[string]any{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
		// Disable chain-of-thought reasoning — not needed for log analysis and wastes tokens.
		"thinking": map[string]any{"type": "disabled"},
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not connect to LLM at %s — is it running? (%w)", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	// Qwen3 and other reasoning models put output in reasoning_content when content is empty.
	content := result.Choices[0].Message.Content
	if content == "" {
		content = result.Choices[0].Message.ReasoningContent
	}
	if content == "" {
		return "", fmt.Errorf("LLM returned empty response")
	}
	return content, nil
}
