package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anamika/zenact-web/server/models"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("OpenRouter returned %d: %s", e.StatusCode, e.Body)
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// --- OpenAI-compatible request/response types ---

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Decide sends the current browser state to the vision LLM and returns a structured action.
func (c *Client) Decide(
	ctx context.Context,
	systemPrompt string,
	screenshot []byte,
	pageURL string,
	pageTitle string,
	taskPrompt string,
	history []models.Step,
) (*models.LLMResponse, error) {
	// System message
	sysMsg := chatMessage{Role: "system", Content: systemPrompt}

	// Build history summary
	historyText := buildHistoryText(history)

	// User message with screenshot + context
	b64Screenshot := base64.StdEncoding.EncodeToString(screenshot)
	userContent := []contentPart{
		{
			Type: "text",
			Text: fmt.Sprintf(
				"Task: %s\n\nCurrent URL: %s\nPage Title: %s\n\nPrevious actions:\n%s\n\nAnalyze the screenshot and decide the next action. Respond with JSON only.",
				taskPrompt, pageURL, pageTitle, historyText,
			),
		},
		{
			Type: "image_url",
			ImageURL: &imageURL{
				URL:    fmt.Sprintf("data:image/png;base64,%s", b64Screenshot),
				Detail: "high",
			},
		},
	}

	userMsg := chatMessage{Role: "user", Content: userContent}

	reqBody := chatRequest{
		Model:    c.model,
		Messages: []chatMessage{sysMsg, userMsg},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openRouterURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://zenact-web.local")
	req.Header.Set("X-Title", "Zenact Web Agent")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("OpenRouter error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := chatResp.Choices[0].Message.Content
	return parseJSONFromContent(content)
}

// parseJSONFromContent extracts JSON from LLM content, handling markdown code fences.
func parseJSONFromContent(content string) (*models.LLMResponse, error) {
	content = strings.TrimSpace(content)

	// Strip markdown code fences if present
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	var resp models.LLMResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON: %w\nraw content: %s", err, content)
	}
	return &resp, nil
}

// buildHistoryText creates a concise summary of previous steps.
func buildHistoryText(history []models.Step) string {
	if len(history) == 0 {
		return "(none — this is the first step)"
	}

	// Only last 5 steps to save tokens
	start := 0
	if len(history) > 5 {
		start = len(history) - 5
	}

	var sb strings.Builder
	for _, step := range history[start:] {
		// Status indicator (SUCCESS/FAILED)
		status := "SUCCESS"
		if !step.ExecutionSuccess {
			status = "FAILED"
		}

		// Build step line
		fmt.Fprintf(&sb, "Step %d [%s]: %s", step.Iteration, status, step.Action.Type)

		if step.Action.Selector != "" {
			fmt.Fprintf(&sb, " on selector=%q", step.Action.Selector)
		}
		if step.Action.Value != "" {
			fmt.Fprintf(&sb, " value=%q", step.Action.Value)
		}

		fmt.Fprintf(&sb, " | URL: %s", step.URL)

		if step.Thought != "" {
			fmt.Fprintf(&sb, " | Thought: %s", step.Thought)
		}

		// Show execution error if failed
		if !step.ExecutionSuccess && step.ExecutionError != "" {
			fmt.Fprintf(&sb, "\n  Execution Error: %s", step.ExecutionError)
		}

		sb.WriteString("\n")
	}
	return sb.String()
}
