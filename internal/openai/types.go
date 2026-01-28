// Package openai provides a client for the OpenAI Chat Completions API.
package openai

// Message represents a single message in a conversation.
type Message struct {
	Role    string `json:"role"`    // "system", "user", or "assistant"
	Content string `json:"content"` // The message content
}

// ChatRequest represents a request to the chat completions endpoint.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ChatResponse represents a response from the chat completions endpoint.
type ChatResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a single completion choice in the response.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage contains token usage information for the request.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// APIError represents an error response from the OpenAI API.
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// Error implements the error interface for APIError.
func (e *APIError) Error() string {
	return e.Message
}

// ErrorResponse wraps an API error for JSON unmarshaling.
type ErrorResponse struct {
	Error APIError `json:"error"`
}
