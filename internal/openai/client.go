// Package openai provides a client for the OpenAI Chat Completions API.
package openai

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen-models.yaml openapi.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen-client.yaml openapi.yaml

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"gopus/internal/config"
)

// ChatClient wraps the generated OpenAI client with configuration defaults.
type ChatClient struct {
	client      *ClientWithResponses
	model       string
	maxTokens   int
	temperature float32
}

// NewChatClient creates a new OpenAI chat client from the provided configuration.
func NewChatClient(cfg *config.Config) (*ChatClient, error) {
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Create request editor to add authorization header
	authEditor := WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+cfg.OpenAI.APIKey)
		return nil
	})

	// Create the generated client
	client, err := NewClientWithResponses(
		cfg.OpenAI.BaseURL,
		WithHTTPClient(httpClient),
		authEditor,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &ChatClient{
		client:      client,
		model:       cfg.OpenAI.Model,
		maxTokens:   cfg.OpenAI.MaxTokens,
		temperature: float32(cfg.OpenAI.Temperature),
	}, nil
}

// RoleUser is the role constant for user messages.
const RoleUser = ChatCompletionRequestMessageRoleUser

// RoleAssistant is the role constant for assistant messages.
const RoleAssistant = ChatCompletionRequestMessageRoleAssistant

// RoleSystem is the role constant for system messages.
const RoleSystem = ChatCompletionRequestMessageRoleSystem

// ChatCompletion sends a chat completion request to the OpenAI API.
func (c *ChatClient) ChatCompletion(ctx context.Context, messages []ChatCompletionRequestMessage) (*ChatCompletionResponse, error) {
	// Build the request
	req := CreateChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   &c.maxTokens,
		Temperature: &c.temperature,
	}

	// Send the request using the generated client
	resp, err := c.client.CreateChatCompletionWithResponse(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Handle error responses
	if resp.JSON400 != nil {
		return nil, &resp.JSON400.Error
	}
	if resp.JSON401 != nil {
		return nil, &resp.JSON401.Error
	}
	if resp.JSON429 != nil {
		return nil, &resp.JSON429.Error
	}
	if resp.JSON500 != nil {
		return nil, &resp.JSON500.Error
	}

	// Check for successful response
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response status: %s", resp.Status())
	}

	return resp.JSON200, nil
}

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	return e.Message
}
