// Package main provides a simple CLI chat application using the OpenAI API.
package main

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

	// Load existing messages into OpenAI format
	chatHistory := history.ConvertSessionMessages(historyManager.Current())

	// Run the chat loop
	runChatLoop(ctx, scanner, client, historyManager, chatHistory)

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}

// runChatLoop runs the main chat loop, reading user input and sending requests to OpenAI.
func runChatLoop(ctx context.Context, scanner *bufio.Scanner, client *openai.ChatClient, historyManager *history.Manager, chatHistory []openai.ChatCompletionRequestMessage) {
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

		// Add user message to history manager (auto-saves)
		if err := historyManager.AddMessage("user", input); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving message: %v\n", err)
		}

		// Add user message to chat history for API
		chatHistory = append(chatHistory, openai.ChatCompletionRequestMessage{
			Role:    openai.RoleUser,
			Content: input,
		})

		// Send request to OpenAI
		resp, err := client.ChatCompletion(ctx, chatHistory)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			// Remove the failed message from both histories
			chatHistory = chatHistory[:len(chatHistory)-1]
			// Remove from session history too
			session := historyManager.Current()
			if len(session.Messages) > 0 {
				session.Messages = session.Messages[:len(session.Messages)-1]
				historyManager.SaveCurrent()
			}
			continue
		}

		// Extract and display the assistant's response
		if len(resp.Choices) == 0 {
			fmt.Fprintln(os.Stderr, "Error: No response from API")
			chatHistory = chatHistory[:len(chatHistory)-1]
			session := historyManager.Current()
			if len(session.Messages) > 0 {
				session.Messages = session.Messages[:len(session.Messages)-1]
				historyManager.SaveCurrent()
			}
			continue
		}

		assistantContent := resp.Choices[0].Message.Content
		if assistantContent == nil {
			fmt.Fprintln(os.Stderr, "Error: Empty response from API")
			chatHistory = chatHistory[:len(chatHistory)-1]
			session := historyManager.Current()
			if len(session.Messages) > 0 {
				session.Messages = session.Messages[:len(session.Messages)-1]
				historyManager.SaveCurrent()
			}
			continue
		}

		assistantMessage := *assistantContent
		printer.PrintMessage("assistant", assistantMessage, false)
		fmt.Println()

		// Add assistant response to history manager (auto-saves)
		if err := historyManager.AddMessage("assistant", assistantMessage); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving message: %v\n", err)
		}

		// Add assistant response to chat history for API
		chatHistory = append(chatHistory, openai.ChatCompletionRequestMessage{
			Role:    openai.RoleAssistant,
			Content: assistantMessage,
		})
	}
}
