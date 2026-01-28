package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gopus/internal/config"
)

// Client is an HTTP client for the OpenAI API.
type Client struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
}

// NewClient creates a new OpenAI API client from the provided configuration.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		apiKey:      cfg.OpenAI.APIKey,
		baseURL:     cfg.OpenAI.BaseURL,
		model:       cfg.OpenAI.Model,
		maxTokens:   cfg.OpenAI.MaxTokens,
		temperature: cfg.OpenAI.Temperature,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ChatCompletion sends a chat completion request to the OpenAI API.
func (c *Client) ChatCompletion(ctx context.Context, messages []Message) (*ChatResponse, error) {
	req := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, &errResp.Error
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API-level errors in the response
	if chatResp.Error != nil {
		return nil, chatResp.Error
	}

	return &chatResp, nil
}
