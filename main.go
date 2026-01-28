// Package main provides a simple CLI chat application using the OpenAI API.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"gopus/internal/config"
	"gopus/internal/openai"
)

func main() {
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nGoodbye!")
		cancel()
		os.Exit(0)
	}()

	// Load configuration
	fmt.Println("Loading configuration from config.yaml...")
	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please copy config.example.yaml to config.yaml and add your API key.")
		os.Exit(1)
	}

	// Create OpenAI client
	client := openai.NewClient(cfg)
	fmt.Printf("Connected to OpenAI (model: %s). Type 'quit' or 'exit' to end the conversation.\n\n", cfg.OpenAI.Model)

	// Initialize conversation history
	var history []openai.Message

	// Create scanner for reading user input
	scanner := bufio.NewScanner(os.Stdin)

	// Main chat loop
	for {
		fmt.Print("You: ")

		// Read user input
		if !scanner.Scan() {
			// EOF or error
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Check for empty input
		if input == "" {
			continue
		}

		// Check for exit commands
		lowerInput := strings.ToLower(input)
		if lowerInput == "quit" || lowerInput == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		// Add user message to history
		history = append(history, openai.Message{
			Role:    "user",
			Content: input,
		})

		// Send request to OpenAI
		resp, err := client.ChatCompletion(ctx, history)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			// Remove the failed message from history
			history = history[:len(history)-1]
			continue
		}

		// Extract and display the assistant's response
		if len(resp.Choices) == 0 {
			fmt.Fprintln(os.Stderr, "Error: No response from API")
			history = history[:len(history)-1]
			continue
		}

		assistantMessage := resp.Choices[0].Message.Content
		fmt.Printf("Assistant: %s\n\n", assistantMessage)

		// Add assistant response to history
		history = append(history, openai.Message{
			Role:    "assistant",
			Content: assistantMessage,
		})
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
