// Package main provides a simple CLI chat application using the OpenAI API.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"gopus/internal/config"
	"gopus/internal/history"
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
	if err := selectSession(historyManager, scanner, cfg.History.TruncateDisplay); err != nil {
		fmt.Fprintf(os.Stderr, "Error selecting session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nConnected to OpenAI (model: %s).\n", cfg.OpenAI.Model)
	fmt.Println("Type 'quit' or 'exit' to end. Type '/help' for commands.\n")

	// Load existing messages into OpenAI format
	chatHistory := loadChatHistory(historyManager.Current())

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

		// Check for session commands
		if strings.HasPrefix(input, "/") {
			handled, newHistory := handleCommand(input, historyManager, scanner)
			if handled {
				if newHistory != nil {
					chatHistory = newHistory
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
		fmt.Printf("Assistant: %s\n\n", assistantMessage)

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

// selectSession displays available sessions and lets the user choose one or create a new one.
// truncateDisplay controls message truncation: 0 = no truncation, >0 = max characters.
func selectSession(manager *history.Manager, scanner *bufio.Scanner, truncateDisplay int) error {
	sessions, err := manager.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No existing sessions found. Starting a new session.")
		manager.NewSession()
		return nil
	}

	fmt.Println("\n=== Available Sessions ===")
	fmt.Println("  0. Start a new session")
	for i, session := range sessions {
		name := session.Name
		if name == "" {
			name = "(unnamed)"
		}
		msgCount := len(session.Messages)
		updated := session.UpdatedAt.Format("2006-01-02 15:04")
		fmt.Printf("  %d. %s (%d messages, last updated: %s)\n", i+1, name, msgCount, updated)
	}
	fmt.Println()

	for {
		fmt.Print("Select a session (0 for new, or number): ")
		if !scanner.Scan() {
			return fmt.Errorf("failed to read input")
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		num, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Please enter a valid number.")
			continue
		}

		if num == 0 {
			fmt.Println("Starting a new session.")
			manager.NewSession()
			return nil
		}

		if num < 1 || num > len(sessions) {
			fmt.Printf("Please enter a number between 0 and %d.\n", len(sessions))
			continue
		}

		selectedSession := sessions[num-1]
		manager.SetCurrent(selectedSession)
		fmt.Printf("Continuing session: %s\n", selectedSession.Name)

		// Display recent messages from the session
		if len(selectedSession.Messages) > 0 {
			fmt.Println("\n--- Recent messages ---")
			start := 0
			if len(selectedSession.Messages) > 6 {
				start = len(selectedSession.Messages) - 6
				fmt.Printf("... (%d earlier messages)\n", start)
			}
			for _, msg := range selectedSession.Messages[start:] {
				role := "You"
				if msg.Role == "assistant" {
					role = "Assistant"
				}
				// Truncate long messages for display if configured
				content := msg.Content
				if truncateDisplay > 0 && len(content) > truncateDisplay {
					content = content[:truncateDisplay] + "..."
				}
				fmt.Printf("%s: %s\n", role, content)
			}
			fmt.Println("--- End of history ---")
			fmt.Printf("\n(Loaded %d messages from history - the AI will have full context)\n", len(selectedSession.Messages))
		}

		return nil
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

// handleCommand processes session management commands.
// Returns true if the command was handled, and optionally a new chat history if session changed.
func handleCommand(input string, manager *history.Manager, scanner *bufio.Scanner) (bool, []openai.ChatCompletionRequestMessage) {
	parts := strings.SplitN(input, " ", 2)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		printHelp()
		return true, nil

	case "/new":
		manager.NewSession()
		fmt.Println("Started a new session.")
		return true, []openai.ChatCompletionRequestMessage{}

	case "/list":
		sessions, err := manager.ListSessions()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
			return true, nil
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return true, nil
		}
		fmt.Println("\n=== Sessions ===")
		currentID := ""
		if manager.Current() != nil {
			currentID = manager.Current().ID
		}
		for i, session := range sessions {
			name := session.Name
			if name == "" {
				name = "(unnamed)"
			}
			marker := ""
			if session.ID == currentID {
				marker = " (current)"
			}
			fmt.Printf("  %d. %s%s\n", i+1, name, marker)
		}
		fmt.Println()
		return true, nil

	case "/switch":
		sessions, err := manager.ListSessions()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
			return true, nil
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions to switch to.")
			return true, nil
		}

		fmt.Println("\n=== Sessions ===")
		for i, session := range sessions {
			name := session.Name
			if name == "" {
				name = "(unnamed)"
			}
			fmt.Printf("  %d. %s\n", i+1, name)
		}
		fmt.Print("Select session number: ")
		if !scanner.Scan() {
			return true, nil
		}
		numStr := strings.TrimSpace(scanner.Text())
		num, err := strconv.Atoi(numStr)
		if err != nil || num < 1 || num > len(sessions) {
			fmt.Println("Invalid selection.")
			return true, nil
		}
		manager.SetCurrent(sessions[num-1])
		fmt.Printf("Switched to session: %s\n", sessions[num-1].Name)
		if len(sessions[num-1].Messages) > 0 {
			fmt.Printf("(Loaded %d messages from history - the AI will have full context)\n", len(sessions[num-1].Messages))
		}
		return true, loadChatHistory(sessions[num-1])

	case "/rename":
		if len(parts) < 2 {
			fmt.Println("Usage: /rename <new name>")
			return true, nil
		}
		newName := strings.TrimSpace(parts[1])
		if manager.Current() == nil {
			fmt.Println("No current session.")
			return true, nil
		}
		manager.Current().Name = newName
		if err := manager.SaveCurrent(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving session: %v\n", err)
			return true, nil
		}
		fmt.Printf("Session renamed to: %s\n", newName)
		return true, nil

	case "/delete":
		if manager.Current() == nil {
			fmt.Println("No current session to delete.")
			return true, nil
		}
		fmt.Printf("Delete session '%s'? (yes/no): ", manager.Current().Name)
		if !scanner.Scan() {
			return true, nil
		}
		confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if confirm != "yes" && confirm != "y" {
			fmt.Println("Deletion cancelled.")
			return true, nil
		}
		if err := manager.DeleteSession(manager.Current().ID); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting session: %v\n", err)
			return true, nil
		}
		fmt.Println("Session deleted. Starting a new session.")
		manager.NewSession()
		return true, []openai.ChatCompletionRequestMessage{}

	case "/info":
		session := manager.Current()
		if session == nil {
			fmt.Println("No current session.")
			return true, nil
		}
		fmt.Printf("\n=== Session Info ===\n")
		fmt.Printf("ID: %s\n", session.ID)
		fmt.Printf("Name: %s\n", session.Name)
		fmt.Printf("Created: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated: %s\n", session.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Messages: %d\n\n", len(session.Messages))
		return true, nil

	default:
		// Not a recognized command
		return false, nil
	}
}

// printHelp displays available commands.
func printHelp() {
	fmt.Println(`
=== Commands ===
  /help    - Show this help message
  /new     - Start a new session
  /list    - List all sessions
  /switch  - Switch to a different session
  /rename  - Rename current session (/rename <name>)
  /delete  - Delete current session
  /info    - Show current session info
  quit     - Exit the application
  exit     - Exit the application
`)
}
