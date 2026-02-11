// Package history provides session management for persistent chat history.
package history

import (
	"time"

	"gopus/internal/openai"
)

// Role represents the role of a message author.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// MessageType distinguishes between regular messages and summaries.
type MessageType string

const (
	TypeMessage MessageType = "message"
	TypeSummary MessageType = "summary"
)

// SummaryLevel indicates the compression level of a summary.
type SummaryLevel string

const (
	LevelCondensed  SummaryLevel = "condensed"  // Medium compression
	LevelCompressed SummaryLevel = "compressed" // High compression
)

// ToolCall represents a tool call made by the assistant.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Message represents a chat message or summary in the history.
type Message struct {
	Role         Role         `json:"role"`
	Content      string       `json:"content"`
	Type         MessageType  `json:"type,omitempty"`          // message or summary (empty defaults to message)
	SummaryLevel SummaryLevel `json:"summary_level,omitempty"` // only for summaries
	MessageCount int          `json:"message_count,omitempty"` // number of messages summarized
	CreatedAt    time.Time    `json:"created_at,omitempty"`

	// Tool-related fields
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // for assistant messages with tool calls
	ToolCallID string     `json:"tool_call_id,omitempty"` // for tool result messages
}

// IsSummary returns true if this message is a summary.
func (m Message) IsSummary() bool {
	return m.Type == TypeSummary
}

// IsMessage returns true if this message is a regular message (not a summary).
func (m Message) IsMessage() bool {
	return m.Type == "" || m.Type == TypeMessage
}

// ToOpenAI converts a Message to the OpenAI API message format.
func (m Message) ToOpenAI() openai.ChatCompletionRequestMessage {
	content := m.Content
	msg := openai.ChatCompletionRequestMessage{
		Role:    openai.ChatCompletionRequestMessageRole(m.Role),
		Content: &content,
	}

	// Handle tool calls (for assistant messages)
	if len(m.ToolCalls) > 0 {
		toolCalls := make([]openai.ChatCompletionMessageToolCall, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			toolCalls[i] = openai.ChatCompletionMessageToolCall{
				Id:   tc.ID,
				Type: openai.ChatCompletionMessageToolCallTypeFunction,
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			}
		}
		msg.ToolCalls = &toolCalls
	}

	// Handle tool result (for tool messages)
	if m.Role == RoleTool && m.ToolCallID != "" {
		msg.ToolCallId = &m.ToolCallID
	}

	return msg
}

// MessageFromOpenAI creates a Message from an OpenAI API message.
func MessageFromOpenAI(msg openai.ChatCompletionRequestMessage) Message {
	content := ""
	if msg.Content != nil {
		content = *msg.Content
	}

	m := Message{
		Role:    Role(msg.Role),
		Content: content,
	}

	// Handle tool calls
	if msg.ToolCalls != nil && len(*msg.ToolCalls) > 0 {
		m.ToolCalls = make([]ToolCall, len(*msg.ToolCalls))
		for i, tc := range *msg.ToolCalls {
			m.ToolCalls[i] = ToolCall{
				ID:        tc.Id,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}
	}

	// Handle tool call ID
	if msg.ToolCallId != nil {
		m.ToolCallID = *msg.ToolCallId
	}

	return m
}

// MessagesToOpenAI converts a slice of Messages to OpenAI API format.
func MessagesToOpenAI(messages []Message) []openai.ChatCompletionRequestMessage {
	result := make([]openai.ChatCompletionRequestMessage, len(messages))
	for i, m := range messages {
		result[i] = m.ToOpenAI()
	}
	return result
}

// MessagesFromOpenAI converts a slice of OpenAI API messages to Messages.
func MessagesFromOpenAI(messages []openai.ChatCompletionRequestMessage) []Message {
	result := make([]Message, len(messages))
	for i, m := range messages {
		result[i] = MessageFromOpenAI(m)
	}
	return result
}
