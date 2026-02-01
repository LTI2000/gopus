// Package history provides session management for persistent chat history.
package history

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"gopus/internal/printer"
	"gopus/internal/table"
)

// buildSessionTable creates a table displaying the given sessions.
func buildSessionTable(sessions []*Session) *table.Table {
	tbl := table.New(
		table.Column{Header: "#", MinWidth: 3, Align: table.AlignLeft},
		table.Column{Header: "Name", MinWidth: 4, MaxWidth: 40, Align: table.AlignLeft},
		table.Column{Header: "Msgs", MinWidth: 4, Align: table.AlignRight},
		table.Column{Header: "Last Updated", Align: table.AlignLeft},
	)

	for i, session := range sessions {
		name := session.Name
		if name == "" {
			name = "(unnamed)"
		}
		msgCount := fmt.Sprintf("%d", len(session.Messages))
		updated := session.UpdatedAt.Format("2006-01-02 15:04")
		tbl.AddRow(fmt.Sprintf("%d", i+1), name, msgCount, updated)
	}

	return tbl
}

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

	// Print table with highlighted first column (row numbers in yellow)
	opts := table.DefaultPrintOptions()
	opts.HighlightColumn = 0
	buildSessionTable(sessions).Print(opts)

	// Determine default selection based on number of sessions
	// If there are saved sessions, default to the most recent one (1)
	// Otherwise, default to creating a new session (0)
	defaultSelection := "0"
	if len(sessions) > 0 {
		defaultSelection = "1"
	}

	for {
		fmt.Printf("Select a session (0 for new, d# to delete, or number) [%s]: ", defaultSelection)
		if !scanner.Scan() {
			return fmt.Errorf("failed to read input")
		}

		input := strings.TrimSpace(scanner.Text())

		// Use default selection when pressing return
		if input == "" {
			input = defaultSelection
		}

		// Handle delete command (d followed by number)
		if after, ok := strings.CutPrefix(strings.ToLower(input), "d"); ok {
			numStr := after
			num, err := strconv.Atoi(numStr)
			if err != nil || num < 1 || num > len(sessions) {
				fmt.Printf("Please enter d followed by a number between 1 and %d.\n", len(sessions))
				continue
			}

			sessionToDelete := sessions[num-1]
			// Confirm deletion
			fmt.Printf("Delete session '%s'? (y/N): ", sessionToDelete.Name)
			if !scanner.Scan() {
				return fmt.Errorf("failed to read input")
			}
			confirm := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if confirm == "y" || confirm == "yes" {
				if err := manager.DeleteSession(sessionToDelete.ID); err != nil {
					fmt.Printf("Failed to delete session: %v\n", err)
				} else {
					fmt.Println("Session deleted.")
					// Remove from local list
					sessions = append(sessions[:num-1], sessions[num:]...)
					if len(sessions) == 0 {
						fmt.Println("No sessions remaining. Starting a new session.")
						manager.NewSession()
						return nil
					}
					// Reprint the table
					buildSessionTable(sessions).Print(opts)
				}
			}
			continue
		}

		num, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Please enter a valid number or d# to delete.")
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
		for _, msg := range selectedSession.Messages {
			printer.PrintMessage(string(msg.Role), msg.Content, true)
		}

		return nil
	}
}
