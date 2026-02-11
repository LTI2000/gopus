package history

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"gopus/internal/openai"
)

// Generate implements quick.Generator for Message.
func (Message) Generate(r *rand.Rand, size int) reflect.Value {
	roles := []Role{RoleUser, RoleAssistant, RoleSystem}
	types := []MessageType{"", TypeMessage, TypeSummary}

	content, _ := quick.Value(reflect.TypeOf(""), r)

	m := Message{
		Role:    roles[r.Intn(len(roles))],
		Content: content.String(),
		Type:    types[r.Intn(len(types))],
	}

	return reflect.ValueOf(m)
}

// TestMessageRoundTrip verifies that converting to OpenAI format and back
// preserves the essential fields (Role and Content).
func TestMessageRoundTrip(t *testing.T) {
	property := func(content string, roleIdx uint8) bool {
		roles := []Role{RoleUser, RoleAssistant, RoleSystem}
		role := roles[int(roleIdx)%len(roles)]

		original := Message{
			Role:    role,
			Content: content,
		}

		// Convert to OpenAI and back
		openaiMsg := original.ToOpenAI()
		restored := MessageFromOpenAI(openaiMsg)

		// Property: Role and Content are preserved
		return restored.Role == original.Role &&
			restored.Content == original.Content
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestMessagesToOpenAILengthPreservation verifies that the output slice
// length equals the input slice length.
func TestMessagesToOpenAILengthPreservation(t *testing.T) {
	property := func(count uint8) bool {
		// Create a slice of messages
		messages := make([]Message, int(count))
		for i := range messages {
			messages[i] = Message{
				Role:    RoleUser,
				Content: "test",
			}
		}

		// Convert to OpenAI format
		result := MessagesToOpenAI(messages)

		// Property: Length is preserved
		return len(result) == len(messages)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestMessagesFromOpenAILengthPreservation verifies that the output slice
// length equals the input slice length.
func TestMessagesFromOpenAILengthPreservation(t *testing.T) {
	property := func(count uint8) bool {
		// Create a slice of OpenAI messages
		messages := make([]openai.ChatCompletionRequestMessage, int(count))
		for i := range messages {
			messages[i] = openai.ChatCompletionRequestMessage{
				Role:    openai.ChatCompletionRequestMessageRoleUser,
				Content: "test",
			}
		}

		// Convert from OpenAI format
		result := MessagesFromOpenAI(messages)

		// Property: Length is preserved
		return len(result) == len(messages)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestMessageTypeMutualExclusion verifies that IsSummary and IsMessage
// behave correctly for known message types.
func TestMessageTypeMutualExclusion(t *testing.T) {
	property := func(typeIdx uint8) bool {
		// Only test known types - empty, TypeMessage, TypeSummary
		types := []MessageType{"", TypeMessage, TypeSummary}
		msgType := types[int(typeIdx)%len(types)]

		m := Message{Type: msgType}

		isSummary := m.IsSummary()
		isMessage := m.IsMessage()

		// Property: For known types, exactly one of IsSummary or IsMessage is true
		if msgType == TypeSummary {
			return isSummary && !isMessage
		}
		// For empty or TypeMessage, IsMessage should be true and IsSummary false
		return !isSummary && isMessage
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestIsSummaryOnlyForTypeSummary verifies that IsSummary returns true
// only when Type is TypeSummary.
func TestIsSummaryOnlyForTypeSummary(t *testing.T) {
	property := func(m Message) bool {
		// Property: IsSummary is true iff Type == TypeSummary
		return m.IsSummary() == (m.Type == TypeSummary)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestIsMessageForNonSummary verifies that IsMessage returns true
// for empty type, TypeMessage, and any unknown type.
func TestIsMessageForNonSummary(t *testing.T) {
	property := func(m Message) bool {
		// Property: IsMessage is true iff Type is not TypeSummary
		return m.IsMessage() == (m.Type == "" || m.Type == TypeMessage)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestToOpenAIRoleMapping verifies that Role is correctly mapped to OpenAI format.
func TestToOpenAIRoleMapping(t *testing.T) {
	property := func(content string, roleIdx uint8) bool {
		roles := []Role{RoleUser, RoleAssistant, RoleSystem}
		role := roles[int(roleIdx)%len(roles)]

		m := Message{
			Role:    role,
			Content: content,
		}

		openaiMsg := m.ToOpenAI()

		// Property: Role is correctly mapped
		return string(openaiMsg.Role) == string(role)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestToOpenAIContentPreservation verifies that Content is preserved in conversion.
func TestToOpenAIContentPreservation(t *testing.T) {
	property := func(content string) bool {
		m := Message{
			Role:    RoleUser,
			Content: content,
		}

		openaiMsg := m.ToOpenAI()

		// Property: Content is preserved
		return openaiMsg.Content == content
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}
