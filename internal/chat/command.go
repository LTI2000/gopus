package chat

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopus/internal/history"
	"gopus/internal/openai"
	"gopus/internal/spinner"
)

// handleCommand processes slash commands. Returns true if the command was handled.
func (c *ChatLoop) handleCommand(ctx context.Context, input string, chatHistory *[]openai.ChatCompletionRequestMessage) bool {
	cmd := strings.ToLower(strings.TrimPrefix(input, "/"))

	switch cmd {
	case "summarize":
		c.handleSummarize(ctx, chatHistory)
		return true
	case "stats":
		c.handleStats()
		return true
	case "help":
		c.handleHelp()
		return true
	default:
		fmt.Printf("Unknown command: %s (type /help for available commands)\n", input)
		return true
	}
}

// handleSummarize processes the /summarize command.
func (c *ChatLoop) handleSummarize(ctx context.Context, chatHistory *[]openai.ChatCompletionRequestMessage) {
	session := c.historyManager.Current()

	if !c.config.Summarization.Enabled {
		fmt.Println("Summarization is disabled in configuration.")
		return
	}

	if !c.summarizer.NeedsSummarization(session.Messages) {
		fmt.Println("No messages need summarization yet.")
		stats := c.summarizer.GetStats(session.Messages)
		fmt.Printf("Current stats: %d total messages, %d recent (kept in full)\n",
			stats.TotalMessages, stats.RecentMessages)
		return
	}

	// Show what will be summarized
	stats := c.summarizer.GetStats(session.Messages)
	fmt.Printf("Summarizing: %d messages to compress, %d to condense, keeping %d recent\n",
		stats.CompressedCount, stats.CondensedMessages, stats.RecentMessages)

	// Start spinner
	spin := spinner.New()
	spin.Start()

	// Process the session
	newMessages, err := c.summarizer.ProcessSession(ctx, session)

	spin.Stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during summarization: %v\n", err)
		return
	}

	// Update session with summarized messages
	session.Messages = newMessages
	if err := c.historyManager.SaveCurrent(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving session: %v\n", err)
		return
	}

	// Update the chat history for API calls
	*chatHistory = history.MessagesToOpenAI(newMessages)

	// Show results
	newStats := c.summarizer.GetStats(newMessages)
	fmt.Printf("✓ Summarization complete. New message count: %d (was %d)\n",
		newStats.TotalMessages, stats.TotalMessages)
}

// handleStats shows summarization statistics.
func (c *ChatLoop) handleStats() {
	session := c.historyManager.Current()
	stats := c.summarizer.GetStats(session.Messages)

	fmt.Println("\n=== Session Statistics ===")
	fmt.Printf("Total messages:      %d\n", stats.TotalMessages)
	fmt.Printf("Recent (full):       %d\n", stats.RecentMessages)
	fmt.Printf("To condense:         %d\n", stats.CondensedMessages)
	fmt.Printf("To compress:         %d\n", stats.CompressedCount)
	fmt.Printf("Existing summaries:  %d\n", stats.ExistingSummaries)
	fmt.Println()

	if c.config.Summarization.AutoSummarize {
		regularCount := stats.TotalMessages - stats.ExistingSummaries
		fmt.Printf("Auto-summarize threshold: %d (current: %d)\n",
			c.config.Summarization.AutoThreshold, regularCount)
	} else {
		fmt.Println("Auto-summarization: disabled")
	}
	fmt.Println()
}

// handleHelp shows available commands.
func (c *ChatLoop) handleHelp() {
	fmt.Println("\n=== Available Commands ===")
	fmt.Println("/summarize  - Summarize older messages to reduce history size")
	fmt.Println("/stats      - Show session statistics and summarization info")
	fmt.Println("/help       - Show this help message")
	fmt.Println()
}
