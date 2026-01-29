// Package main provides a simple CLI chat application using the OpenAI API.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"gopus/internal/chat"
	"gopus/internal/config"
	"gopus/internal/history"
	"gopus/internal/openai"
	"gopus/internal/signal"
)

func main() {
	// Set up signal handling for graceful shutdown
	signal.RunWithContext(main0)
}

func main0(ctx context.Context) {
	fmt.Printf("Press Ctrl+D to end the session.\n")

	// Load configuration
	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please copy config.example.yaml to config.yaml and add your API key.")
		os.Exit(1)
	}

	// Create OpenAI client
	client, err := openai.NewChatClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Initialize history manager (use configured sessions_dir or default)
	historyManager, err := history.NewManager(cfg.History.SessionsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing history: %v\n", err)
		os.Exit(1)
	}

	// Create scanner for reading user input
	scanner := bufio.NewScanner(os.Stdin)

	// Session selection at startup
	if err := history.SelectSession(historyManager, scanner); err != nil {
		fmt.Fprintf(os.Stderr, "Error selecting session: %v\n", err)
		os.Exit(1)
	}

	// Run the chat loop
	chat.RunLoop(ctx, scanner, client, historyManager)

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
