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
func SelectSession(manager *Manager, scanner *bufio.Scanner) error {
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
				printer.PrintMessage(msg.Role, msg.Content, true)
			}
			fmt.Println()
		}

		return nil
	}
}
