package chat

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopus/internal/history"
	"gopus/internal/openai"
)

// handleCommand processes slash commands. Returns true if the command was handled.
func (c *ChatLoop) handleCommand(ctx context.Context, input string, chatHistory *[]openai.ChatCompletionRequestMessage) bool {
	// Parse command and arguments
	cmdLine := strings.TrimPrefix(input, "/")
	parts := strings.SplitN(cmdLine, " ", 2)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "summarize":
		c.handleSummarize(ctx, chatHistory)
		return true
	case "stats":
		c.handleStats()
		return true
	case "tools":
		c.handleTools()
		return true
	case "servers":
		c.handleServers()
		return true
	case "sleep":
		c.handleSleep(args)
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

	// Process the session with spinner
	newMessages, err := WithSpinner(func() ([]history.Message, error) {
		return c.summarizer.ProcessSession(ctx, session)
	})

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

// handleSleep runs the animation for a specified duration to test it.
func (c *ChatLoop) handleSleep(args string) {
	// Default to 3 seconds if no argument provided
	seconds := 3.0
	if args != "" {
		parsed, err := strconv.ParseFloat(args, 64)
		if err != nil {
			fmt.Printf("Invalid duration: %s (expected number of seconds)\n", args)
			return
		}
		seconds = parsed
	}

	if seconds <= 0 {
		fmt.Println("Duration must be positive")
		return
	}

	if seconds > 60 {
		fmt.Println("Duration capped at 60 seconds")
		seconds = 60
	}

	fmt.Printf("Sleeping for %.1f seconds...\n", seconds)

	// Sleep with spinner animation
	_, _ = WithSpinner(func() (any, error) {
		time.Sleep(time.Duration(seconds * float64(time.Second)))
		return nil, nil
	})

	fmt.Println("Done!")
}

// handleTools shows available MCP tools.
func (c *ChatLoop) handleTools() {
	if c.mcpManager == nil {
		fmt.Println("MCP is not configured.")
		return
	}

	tools := c.mcpManager.ListTools()
	if len(tools) == 0 {
		fmt.Println("No tools available.")
		return
	}

	fmt.Println("\n=== Available Tools ===")
	for _, tool := range tools {
		fmt.Printf("  %s\n", tool.Name)
		if tool.Description != "" {
			fmt.Printf("    %s\n", tool.Description)
		}
	}
	fmt.Printf("\nTotal: %d tool(s)\n\n", len(tools))
}

// handleServers shows connected MCP servers.
func (c *ChatLoop) handleServers() {
	if c.mcpManager == nil {
		fmt.Println("MCP is not configured.")
		return
	}

	serverCount := c.mcpManager.ServerCount()
	if serverCount == 0 {
		fmt.Println("No MCP servers connected.")
		return
	}

	fmt.Println("\n=== Connected MCP Servers ===")
	fmt.Printf("Total: %d server(s) connected\n", serverCount)
	fmt.Printf("Total tools: %d\n\n", c.mcpManager.ToolCount())
}

// handleHelp shows available commands.
func (c *ChatLoop) handleHelp() {
	fmt.Println("\n=== Available Commands ===")
	fmt.Println("/summarize      - Summarize older messages to reduce history size")
	fmt.Println("/stats          - Show session statistics and summarization info")
	fmt.Println("/tools          - List available MCP tools")
	fmt.Println("/servers        - Show connected MCP servers")
	fmt.Println("/sleep [secs]   - Test animation (default: 3 seconds)")
	fmt.Println("/help           - Show this help message")
	fmt.Println()
}
