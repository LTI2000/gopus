// Package chat provides the main chat loop functionality.
package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"gopus/internal/config"
	"gopus/internal/history"
	"gopus/internal/openai"
	"gopus/internal/printer"
	"gopus/internal/spinner"
	"gopus/internal/summarize"
)

// ChatLoop holds the dependencies for the chat loop.
type ChatLoop struct {
	client         *openai.ChatClient
	historyManager *history.Manager
	summarizer     *summarize.Summarizer
	config         *config.Config
}

// NewChatLoop creates a new chat loop with the given dependencies.
func NewChatLoop(client *openai.ChatClient, historyManager *history.Manager, cfg *config.Config) *ChatLoop {
	return &ChatLoop{
		client:         client,
		historyManager: historyManager,
		summarizer:     summarize.New(client, cfg.Summarization),
		config:         cfg,
	}
}

// Run runs the main chat loop, reading user input and sending requests to OpenAI.
func (c *ChatLoop) Run(ctx context.Context, scanner *bufio.Scanner) {
	// Display help at startup
	c.handleHelp()

	// Convert session messages to OpenAI format for API calls
	session := c.historyManager.Current()
	chatHistory := history.MessagesToOpenAI(session.Messages)

	for {
		fmt.Printf("%suser:%s ", printer.ColorGreen, printer.ColorReset)

		// Read user input (Ctrl+D ends the input stream)
		if !scanner.Scan() {
			// EOF (Ctrl+D) or error - exit the loop
			fmt.Println()
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Check for empty input
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if c.handleCommand(ctx, input, &chatHistory) {
				continue
			}
		}

		// Add user message to history manager (auto-saves)
		if err := c.historyManager.AddMessage(history.RoleUser, input); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving message: %v\n", err)
		}

		// Add user message to chat history for API
		chatHistory = append(chatHistory, openai.ChatCompletionRequestMessage{
			Role:    openai.RoleUser,
			Content: input,
		})

		// Start the spinner animation
		spin := spinner.New()
		spin.Start()

		// Send request to OpenAI
		resp, err := c.client.ChatCompletion(ctx, chatHistory)

		// Stop the spinner before showing response or error
		spin.Stop()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			// Remove the failed message from both histories
			chatHistory = chatHistory[:len(chatHistory)-1]
			// Remove from session history too
			session := c.historyManager.Current()
			if len(session.Messages) > 0 {
				session.Messages = session.Messages[:len(session.Messages)-1]
				c.historyManager.SaveCurrent()
			}
			continue
		}

		// Extract and display the assistant's response
		if len(resp.Choices) == 0 {
			fmt.Fprintln(os.Stderr, "Error: No response from API")
			chatHistory = chatHistory[:len(chatHistory)-1]
			session := c.historyManager.Current()
			if len(session.Messages) > 0 {
				session.Messages = session.Messages[:len(session.Messages)-1]
				c.historyManager.SaveCurrent()
			}
			continue
		}

		assistantContent := resp.Choices[0].Message.Content
		if assistantContent == nil {
			fmt.Fprintln(os.Stderr, "Error: Empty response from API")
			chatHistory = chatHistory[:len(chatHistory)-1]
			session := c.historyManager.Current()
			if len(session.Messages) > 0 {
				session.Messages = session.Messages[:len(session.Messages)-1]
				c.historyManager.SaveCurrent()
			}
			continue
		}

		assistantMessage := *assistantContent
		printer.PrintMessage(string(history.RoleAssistant), assistantMessage, false)
		fmt.Println()

		// Add assistant response to history manager (auto-saves)
		if err := c.historyManager.AddMessage(history.RoleAssistant, assistantMessage); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving message: %v\n", err)
		}

		// Add assistant response to chat history for API
		chatHistory = append(chatHistory, openai.ChatCompletionRequestMessage{
			Role:    openai.RoleAssistant,
			Content: assistantMessage,
		})

		// Check for auto-summarization
		c.checkAutoSummarize(ctx, &chatHistory)
	}
}

// handleCommand processes slash commands. Returns true if the command was handled.
func (c *ChatLoop) handleCommand(ctx context.Context, input string, chatHistory *[]openai.ChatCompletionRequestMessage) bool {
	cmd := strings.ToLower(strings.TrimPrefix(input, "/"))

	switch {
	case cmd == "summarize":
		c.handleSummarize(ctx, chatHistory)
		return true
	case cmd == "stats":
		c.handleStats()
		return true
	case cmd == "help":
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

// checkAutoSummarize checks if auto-summarization should be triggered.
func (c *ChatLoop) checkAutoSummarize(ctx context.Context, chatHistory *[]openai.ChatCompletionRequestMessage) {
	session := c.historyManager.Current()

	if !c.summarizer.ShouldAutoSummarize(session.Messages) {
		return
	}

	fmt.Println("\n[Auto-summarizing history...]")

	// Start spinner
	spin := spinner.New()
	spin.Start()

	// Process the session
	newMessages, err := c.summarizer.ProcessSession(ctx, session)

	spin.Stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Auto-summarization error: %v\n", err)
		return
	}

	// Update session with summarized messages
	oldCount := len(session.Messages)
	session.Messages = newMessages
	if err := c.historyManager.SaveCurrent(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving session: %v\n", err)
		return
	}

	// Update the chat history for API calls
	*chatHistory = history.MessagesToOpenAI(newMessages)

	fmt.Printf("[✓ Auto-summarized: %d → %d messages]\n\n", oldCount, len(newMessages))
}

// RunLoop is a convenience function for backward compatibility.
// Deprecated: Use NewChatLoop and Run instead.
func RunLoop(ctx context.Context, scanner *bufio.Scanner, client *openai.ChatClient, historyManager *history.Manager) {
	// Create a default config for backward compatibility
	cfg := &config.Config{
		Summarization: config.SummarizationConfig{
			Enabled:        true,
			RecentCount:    20,
			CondensedCount: 50,
			AutoSummarize:  false, // Disable auto for backward compat
			AutoThreshold:  100,
		},
	}
	loop := NewChatLoop(client, historyManager, cfg)
	loop.Run(ctx, scanner)
}
