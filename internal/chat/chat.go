// Package chat provides the main chat loop functionality.
package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"gopus/internal/history"
	"gopus/internal/openai"
	"gopus/internal/printer"
	"gopus/internal/spinner"
)

// RunLoop runs the main chat loop, reading user input and sending requests to OpenAI.
func RunLoop(ctx context.Context, scanner *bufio.Scanner, client *openai.ChatClient, historyManager *history.Manager) {
	// Use session messages directly (they are already openai.ChatCompletionRequestMessage)
	session := historyManager.Current()
	chatHistory := make([]openai.ChatCompletionRequestMessage, len(session.Messages))
	copy(chatHistory, session.Messages)

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
		if err := historyManager.AddMessage(openai.RoleUser, input); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving message: %v\n", err)
		}

		// Add user message to chat history for API
		chatHistory = append(chatHistory, openai.ChatCompletionRequestMessage{
			Role:    openai.RoleUser,
			Content: input,
		})

		// Start the fancy ASCII art spinner animation
		spin := spinner.New(spinner.StyleRobot)
		spin.Start()

		// Send request to OpenAI
		resp, err := client.ChatCompletion(ctx, chatHistory)

		// Stop the spinner before showing response or error
		spin.Stop()

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
		printer.PrintMessage(openai.RoleAssistant, assistantMessage, false)
		fmt.Println()

		// Add assistant response to history manager (auto-saves)
		if err := historyManager.AddMessage(openai.RoleAssistant, assistantMessage); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving message: %v\n", err)
		}

		// Add assistant response to chat history for API
		chatHistory = append(chatHistory, openai.ChatCompletionRequestMessage{
			Role:    openai.RoleAssistant,
			Content: assistantMessage,
		})
	}
}
