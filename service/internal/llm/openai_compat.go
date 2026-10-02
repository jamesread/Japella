package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout   = 60 * time.Second
	maxResponseBytes = 2 << 20 // 2 MiB
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Client calls an OpenAI-compatible Chat Completions endpoint.
type Client struct {
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				if err := validateURL(req.URL.String()); err != nil {
					return err
				}
				return nil
			},
		},
	}
}

func NormalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("base URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid base URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("base URL must use http or https")
	}
	if u.Host == "" {
		return "", fmt.Errorf("base URL must include a host")
	}
	u.Fragment = ""
	normalized := strings.TrimRight(u.String(), "/")
	return normalized, nil
}

func validateURL(raw string) error {
	_, err := NormalizeBaseURL(raw)
	return err
}

func CompletionsURL(baseURL string) (string, error) {
	normalized, err := NormalizeBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/chat/completions", nil
	}
	if strings.HasSuffix(normalized, "/chat/completions") {
		return normalized, nil
	}
	return normalized + "/v1/chat/completions", nil
}

func ModelsURL(baseURL string) (string, error) {
	normalized, err := NormalizeBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/models", nil
	}
	if strings.HasSuffix(normalized, "/models") {
		return normalized, nil
	}
	return normalized + "/v1/models", nil
}

type ModelInfo struct {
	ID   string
	Name string
}

type modelsListResponse struct {
	Data []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) ListModels(ctx context.Context, baseURL, apiKey string) ([]ModelInfo, error) {
	endpoint, err := ModelsURL(baseURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if key := strings.TrimSpace(apiKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client := c.HTTPClient
	if client == nil {
		client = NewClient().HTTPClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm models request failed: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read llm models response: %w", err)
	}
	if len(raw) > maxResponseBytes {
		return nil, fmt.Errorf("llm models response too large")
	}

	var parsed modelsListResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode llm models response (HTTP %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("llm error: %s", parsed.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("llm models HTTP %d", resp.StatusCode)
	}

	out := make([]ModelInfo, 0, len(parsed.Data))
	seen := make(map[string]struct{}, len(parsed.Data))
	for _, m := range parsed.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(m.Name)
		if name == "" {
			name = id
		}
		out = append(out, ModelInfo{ID: id, Name: name})
	}
	return out, nil
}

func (c *Client) ChatCompletion(ctx context.Context, baseURL, apiKey, model string, messages []ChatMessage) (string, error) {
	if strings.TrimSpace(model) == "" {
		return "", fmt.Errorf("model is required")
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("messages are required")
	}
	endpoint, err := CompletionsURL(baseURL)
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(apiKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client := c.HTTPClient
	if client == nil {
		client = NewClient().HTTPClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("read llm response: %w", err)
	}
	if len(raw) > maxResponseBytes {
		return "", fmt.Errorf("llm response too large")
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode llm response (HTTP %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("llm error: %s", parsed.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm HTTP %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("llm returned empty content")
	}
	return content, nil
}

// Ping sends a minimal chat completion to verify the connection.
func (c *Client) Ping(ctx context.Context, baseURL, apiKey, model string) error {
	_, err := c.ChatCompletion(ctx, baseURL, apiKey, model, []ChatMessage{
		{Role: "user", Content: "Respond with the single word: pong"},
	})
	return err
}
