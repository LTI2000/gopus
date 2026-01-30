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

// Message represents a chat message or summary in the history.
type Message struct {
	Role         Role         `json:"role"`
	Content      string       `json:"content"`
	Type         MessageType  `json:"type,omitempty"`          // message or summary (empty defaults to message)
	SummaryLevel SummaryLevel `json:"summary_level,omitempty"` // only for summaries
	MessageCount int          `json:"message_count,omitempty"` // number of messages summarized
	CreatedAt    time.Time    `json:"created_at,omitempty"`
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
	return openai.ChatCompletionRequestMessage{
		Role:    openai.ChatCompletionRequestMessageRole(m.Role),
		Content: m.Content,
	}
}

// MessageFromOpenAI creates a Message from an OpenAI API message.
func MessageFromOpenAI(msg openai.ChatCompletionRequestMessage) Message {
	return Message{
		Role:    Role(msg.Role),
		Content: msg.Content,
	}
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
