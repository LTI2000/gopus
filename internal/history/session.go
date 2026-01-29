// Package history provides session management for persistent chat history.
package history

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"gopus/internal/printer"
)

// SelectSession displays available sessions and lets the user choose one or create a new one.
// truncateDisplay controls message truncation: 0 = no truncation, >0 = max characters.
func SelectSession(manager *Manager, scanner *bufio.Scanner, truncateDisplay int) error {
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

		// Display loaded messages in dim colors to distinguish from new messages
		if len(selectedSession.Messages) > 0 {
			fmt.Println()
			for _, msg := range selectedSession.Messages {
				// Truncate long messages for display if configured
				content := msg.Content
				if truncateDisplay > 0 && len(content) > truncateDisplay {
					content = content[:truncateDisplay] + "..."
				}
				printer.PrintMessage(msg.Role, content, true)
			}
			fmt.Println()
		}

		return nil
	}
}

// CommandResult represents the result of handling a command.
type CommandResult struct {
	Handled        bool
	SessionChanged bool
}

// HandleCommand processes session management commands.
// Returns a CommandResult indicating if the command was handled and if the session changed.
func HandleCommand(input string, manager *Manager, scanner *bufio.Scanner) CommandResult {
	parts := strings.SplitN(input, " ", 2)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		printHelp()
		return CommandResult{Handled: true, SessionChanged: false}

	case "/new":
		manager.NewSession()
		fmt.Println("Started a new session.")
		return CommandResult{Handled: true, SessionChanged: true}

	case "/list":
		sessions, err := manager.ListSessions()
		if err != nil {
			fmt.Printf("Error listing sessions: %v\n", err)
			return CommandResult{Handled: true, SessionChanged: false}
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return CommandResult{Handled: true, SessionChanged: false}
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
		return CommandResult{Handled: true, SessionChanged: false}

	case "/switch":
		sessions, err := manager.ListSessions()
		if err != nil {
			fmt.Printf("Error listing sessions: %v\n", err)
			return CommandResult{Handled: true, SessionChanged: false}
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions to switch to.")
			return CommandResult{Handled: true, SessionChanged: false}
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
			return CommandResult{Handled: true, SessionChanged: false}
		}
		numStr := strings.TrimSpace(scanner.Text())
		num, err := strconv.Atoi(numStr)
		if err != nil || num < 1 || num > len(sessions) {
			fmt.Println("Invalid selection.")
			return CommandResult{Handled: true, SessionChanged: false}
		}
		selectedSession := sessions[num-1]
		manager.SetCurrent(selectedSession)
		fmt.Printf("Switched to session: %s\n", selectedSession.Name)
		// Display loaded messages in dim colors to distinguish from new messages
		if len(selectedSession.Messages) > 0 {
			fmt.Println()
			for _, msg := range selectedSession.Messages {
				printer.PrintMessage(msg.Role, msg.Content, true)
			}
			fmt.Println()
		}
		return CommandResult{Handled: true, SessionChanged: true}

	case "/rename":
		if len(parts) < 2 {
			fmt.Println("Usage: /rename <new name>")
			return CommandResult{Handled: true, SessionChanged: false}
		}
		newName := strings.TrimSpace(parts[1])
		if manager.Current() == nil {
			fmt.Println("No current session.")
			return CommandResult{Handled: true, SessionChanged: false}
		}
		manager.Current().Name = newName
		if err := manager.SaveCurrent(); err != nil {
			fmt.Printf("Error saving session: %v\n", err)
			return CommandResult{Handled: true, SessionChanged: false}
		}
		fmt.Printf("Session renamed to: %s\n", newName)
		return CommandResult{Handled: true, SessionChanged: false}

	case "/delete":
		if manager.Current() == nil {
			fmt.Println("No current session to delete.")
			return CommandResult{Handled: true, SessionChanged: false}
		}
		fmt.Printf("Delete session '%s'? (yes/no): ", manager.Current().Name)
		if !scanner.Scan() {
			return CommandResult{Handled: true, SessionChanged: false}
		}
		confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if confirm != "yes" && confirm != "y" {
			fmt.Println("Deletion cancelled.")
			return CommandResult{Handled: true, SessionChanged: false}
		}
		if err := manager.DeleteSession(manager.Current().ID); err != nil {
			fmt.Printf("Error deleting session: %v\n", err)
			return CommandResult{Handled: true, SessionChanged: false}
		}
		fmt.Println("Session deleted. Starting a new session.")
		manager.NewSession()
		return CommandResult{Handled: true, SessionChanged: true}

	case "/info":
		session := manager.Current()
		if session == nil {
			fmt.Println("No current session.")
			return CommandResult{Handled: true, SessionChanged: false}
		}
		fmt.Printf("\n=== Session Info ===\n")
		fmt.Printf("ID: %s\n", session.ID)
		fmt.Printf("Name: %s\n", session.Name)
		fmt.Printf("Created: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated: %s\n", session.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Messages: %d\n\n", len(session.Messages))
		return CommandResult{Handled: true, SessionChanged: false}

	default:
		// Not a recognized command
		return CommandResult{Handled: false, SessionChanged: false}
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
