// Package history provides session management for persistent chat history.
package history

import "gopus/internal/openai"

// Role represents the role of a message author.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message represents a chat message in the history.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
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
