// Package openai provides a client for the OpenAI Chat Completions API.
package openai

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen-models.yaml openapi.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen-client.yaml openapi.yaml

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gopus/internal/config"
)

// ErrEmptyResponse is returned when the API returns no choices or empty message content.
var ErrEmptyResponse = errors.New("empty response from API")

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
	return c.ChatCompletionWithTools(ctx, messages, nil)
}

// ChatCompletionWithTools sends a chat completion request with optional tools.
func (c *ChatClient) ChatCompletionWithTools(ctx context.Context, messages []ChatCompletionRequestMessage, tools []ChatCompletionTool) (*ChatCompletionResponse, error) {
	// Build the request
	req := CreateChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   &c.maxTokens,
		Temperature: &c.temperature,
	}

	// Add tools if provided
	if len(tools) > 0 {
		req.Tools = &tools
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

// ChatCompletionX is a convenience function that calls ChatCompletion and extracts
// the message content, handling all cases of empty choices or nil message content.
// Returns ErrEmptyResponse if the response has no choices or empty content.
func (c *ChatClient) ChatCompletionX(ctx context.Context, messages []ChatCompletionRequestMessage) (string, error) {
	resp, err := c.ChatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}
	return extractMessageContent(resp)
}

// ExtractMessageContent extracts the message content from a ChatCompletionResponse.
// Returns ErrEmptyResponse if the response has no choices or empty content.
func extractMessageContent(resp *ChatCompletionResponse) (string, error) {
	choice, err := extractFirstChoice(resp)
	if err != nil {
		return "", err
	}
	if choice == nil {
		return "", ErrEmptyResponse
	}
	return *choice.Message.Content, nil
}

// ChatCompletionWithToolsX is a convenience function that calls ChatCompletionWithTools and extracts
// the first choice, handling the case of empty choices.
// Returns ErrEmptyResponse if the response has no choices.
func (c *ChatClient) ChatCompletionWithToolsX(ctx context.Context, messages []ChatCompletionRequestMessage, tools []ChatCompletionTool) (*ChatCompletionChoice, error) {
	resp, err := c.ChatCompletionWithTools(ctx, messages, tools)
	if err != nil {
		return nil, err
	}
	return extractFirstChoice(resp)
}

// ExtractFirstChoice extracts the first choice from a ChatCompletionResponse.
// Returns ErrEmptyResponse if the response has no choices.
func extractFirstChoice(resp *ChatCompletionResponse) (*ChatCompletionChoice, error) {
	if len(resp.Choices) == 0 {
		return nil, ErrEmptyResponse
	}
	return &resp.Choices[0], nil
}
