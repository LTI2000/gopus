// Package summarize provides chat history summarization functionality.
package summarize

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gopus/internal/config"
	"gopus/internal/history"
	"gopus/internal/openai"
)

// Summarizer handles chat history summarization.
type Summarizer struct {
	client *openai.ChatClient
	config config.SummarizationConfig
}

// New creates a new Summarizer with the given client and configuration.
func New(client *openai.ChatClient, cfg config.SummarizationConfig) *Summarizer {
	return &Summarizer{
		client: client,
		config: cfg,
	}
}

// TierClassification holds messages classified by tier.
type TierClassification struct {
	Recent     []history.Message // Messages to keep in full detail
	ToCondense []history.Message // Messages to condense
	ToCompress []history.Message // Messages to highly compress
	Existing   []history.Message // Existing summaries to preserve or re-compress
}

// ClassifyTiers classifies messages into tiers based on configuration.
// Messages are ordered from oldest to newest.
func (s *Summarizer) ClassifyTiers(messages []history.Message) TierClassification {
	var result TierClassification

	// Separate existing summaries from regular messages
	var regularMessages []history.Message
	for _, msg := range messages {
		if msg.IsSummary() {
			result.Existing = append(result.Existing, msg)
		} else {
			regularMessages = append(regularMessages, msg)
		}
	}

	totalRegular := len(regularMessages)
	if totalRegular == 0 {
		return result
	}

	// Calculate tier boundaries (from the end, since recent messages are at the end)
	recentStart := totalRegular - s.config.RecentCount
	if recentStart < 0 {
		recentStart = 0
	}

	condensedStart := recentStart - s.config.CondensedCount
	if condensedStart < 0 {
		condensedStart = 0
	}

	// Classify messages
	for i, msg := range regularMessages {
		switch {
		case i >= recentStart:
			result.Recent = append(result.Recent, msg)
		case i >= condensedStart:
			result.ToCondense = append(result.ToCondense, msg)
		default:
			result.ToCompress = append(result.ToCompress, msg)
		}
	}

	return result
}

// NeedsSummarization returns true if there are messages that need summarization.
func (s *Summarizer) NeedsSummarization(messages []history.Message) bool {
	tiers := s.ClassifyTiers(messages)
	return len(tiers.ToCondense) > 0 || len(tiers.ToCompress) > 0
}

// ShouldAutoSummarize returns true if auto-summarization should be triggered.
func (s *Summarizer) ShouldAutoSummarize(messages []history.Message) bool {
	if !s.config.AutoSummarize {
		return false
	}

	// Count regular messages (not summaries)
	count := 0
	for _, msg := range messages {
		if msg.IsMessage() {
			count++
		}
	}

	return count > s.config.AutoThreshold
}

// SummarizeMessages generates a summary for a group of messages.
func (s *Summarizer) SummarizeMessages(ctx context.Context, messages []history.Message, level history.SummaryLevel) (history.Message, error) {
	if len(messages) == 0 {
		return history.Message{}, fmt.Errorf("no messages to summarize")
	}

	// Build the conversation text
	var conversationBuilder strings.Builder
	for _, msg := range messages {
		conversationBuilder.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
	}

	// Select prompt based on level (using configurable prompts)
	prompt := s.config.CondensedPrompt
	if level == history.LevelCompressed {
		prompt = s.config.CompressedPrompt
	}

	// Create the summarization request
	userContent := conversationBuilder.String()
	apiMessages := []openai.ChatCompletionRequestMessage{
		{
			Role:    openai.RoleSystem,
			Content: &prompt,
		},
		{
			Role:    openai.RoleUser,
			Content: &userContent,
		},
	}

	// Call OpenAI API
	content, err := s.client.ChatCompletionX(ctx, apiMessages)
	if err != nil {
		return history.Message{}, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Create the summary message
	return history.Message{
		Role:         history.RoleSystem,
		Content:      content,
		Type:         history.TypeSummary,
		SummaryLevel: level,
		MessageCount: len(messages),
		CreatedAt:    time.Now(),
	}, nil
}

// ProcessSession summarizes a session's messages according to tier configuration.
// Returns the new message list with summaries replacing original messages.
func (s *Summarizer) ProcessSession(ctx context.Context, session *history.Session) ([]history.Message, error) {
	if !s.config.Enabled {
		return session.Messages, nil
	}

	tiers := s.ClassifyTiers(session.Messages)

	var result []history.Message

	// Process messages that need to be compressed (oldest tier)
	if len(tiers.ToCompress) > 0 {
		// Check if we already have existing compressed summaries
		var existingCompressed []history.Message
		for _, msg := range tiers.Existing {
			if msg.SummaryLevel == history.LevelCompressed {
				existingCompressed = append(existingCompressed, msg)
			}
		}

		// Combine existing compressed summaries with new messages to compress
		toCompressAll := append(existingCompressed, tiers.ToCompress...)

		if len(toCompressAll) > 0 {
			summary, err := s.SummarizeMessages(ctx, toCompressAll, history.LevelCompressed)
			if err != nil {
				return nil, fmt.Errorf("failed to create compressed summary: %w", err)
			}
			result = append(result, summary)
		}
	} else {
		// Keep existing compressed summaries
		for _, msg := range tiers.Existing {
			if msg.SummaryLevel == history.LevelCompressed {
				result = append(result, msg)
			}
		}
	}

	// Process messages that need to be condensed
	if len(tiers.ToCondense) > 0 {
		summary, err := s.SummarizeMessages(ctx, tiers.ToCondense, history.LevelCondensed)
		if err != nil {
			return nil, fmt.Errorf("failed to create condensed summary: %w", err)
		}
		result = append(result, summary)
	} else {
		// Keep existing condensed summaries if no new condensing needed
		for _, msg := range tiers.Existing {
			if msg.SummaryLevel == history.LevelCondensed {
				result = append(result, msg)
			}
		}
	}

	// Keep recent messages in full
	result = append(result, tiers.Recent...)

	return result, nil
}

// Stats returns summarization statistics for a session.
type Stats struct {
	TotalMessages     int
	RecentMessages    int
	CondensedMessages int
	CompressedCount   int
	ExistingSummaries int
}

// GetStats returns statistics about how messages would be classified.
func (s *Summarizer) GetStats(messages []history.Message) Stats {
	tiers := s.ClassifyTiers(messages)
	return Stats{
		TotalMessages:     len(messages),
		RecentMessages:    len(tiers.Recent),
		CondensedMessages: len(tiers.ToCondense),
		CompressedCount:   len(tiers.ToCompress),
		ExistingSummaries: len(tiers.Existing),
	}
}
