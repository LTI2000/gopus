// Package printer provides terminal output formatting with ANSI colors for chat messages.
package printer

import (
	"fmt"

	"gopus/internal/openai"
)

// ANSI color codes for terminal output
const (
	ColorReset = "\033[0m"

	// Bright colors for new messages
	ColorGreen = "\033[32m" // Green for user messages
	ColorBlue  = "\033[34m" // Blue for assistant messages

	// Dim colors for loaded/historical messages
	ColorDim      = "\033[2m"    // Dim/faint text for loaded messages
	ColorDimGreen = "\033[2;32m" // Dim green for loaded user messages
	ColorDimBlue  = "\033[2;34m" // Dim blue for loaded assistant messages
)

// PrintMessage outputs a chat message with appropriate formatting based on role and history status.
// role: the message role (user, assistant, or system)
// message: the content to display
// isHistory: if true, uses dim colors for historical/loaded messages
func PrintMessage(role openai.ChatCompletionRequestMessageRole, message string, isHistory bool) {
	var roleColor, messageColor string
	if role == openai.RoleAssistant {
		if isHistory {
			roleColor = ColorDimBlue
		} else {
			roleColor = ColorBlue
		}
	} else {
		if isHistory {
			roleColor = ColorDimGreen
		} else {
			roleColor = ColorGreen
		}
	}
	if isHistory {
		messageColor = ColorDim
	} else {
		messageColor = ColorReset
	}
	fmt.Printf("%s%s%s: %s%s%s\n", roleColor, role, ColorReset, messageColor, message, ColorReset)
}
