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
	fmt.Printf("Type 'quit' or 'exit' to end. Type '/help' for commands.\n")

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
	chatHistory := loadChatHistory(historyManager.Current())

	// Main chat loop
	for {
		fmt.Printf("%suser:%s ", printer.ColorGreen, printer.ColorReset)

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

		// Check for session commands
		if strings.HasPrefix(input, "/") {
			result := history.HandleCommand(input, historyManager, scanner)
			if result.Handled {
				if result.SessionChanged {
					chatHistory = loadChatHistory(historyManager.Current())
				}
				continue
			}
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

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}

// loadChatHistory converts session messages to OpenAI chat format.
func loadChatHistory(session *history.Session) []openai.ChatCompletionRequestMessage {
	if session == nil {
		return nil
	}

	messages := make([]openai.ChatCompletionRequestMessage, 0, len(session.Messages))
	for _, msg := range session.Messages {
		var role openai.ChatCompletionRequestMessageRole
		switch msg.Role {
		case "user":
			role = openai.RoleUser
		case "assistant":
			role = openai.RoleAssistant
		case "system":
			role = openai.RoleSystem
		default:
			role = openai.RoleUser
		}
		messages = append(messages, openai.ChatCompletionRequestMessage{
			Role:    role,
			Content: msg.Content,
		})
	}
	return messages
}
