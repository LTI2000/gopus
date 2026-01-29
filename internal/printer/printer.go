// Package printer provides terminal output formatting with ANSI colors for chat messages.
package printer

import (
	"fmt"

	"gopus/internal/openai"
)

// ANSI escape codes for terminal output
const (
	ColorReset = "\033[0m"
	ColorDim   = "\033[2m" // Dim/faint intensity
	ColorGreen = "\033[32m"
	ColorBlue  = "\033[34m"
)

// PrintMessage outputs a chat message with appropriate formatting based on role and history status.
// role: the message role (user, assistant, or system)
// message: the content to display
// isHistory: if true, uses dim intensity for historical/loaded messages
func PrintMessage(role openai.ChatCompletionRequestMessageRole, message string, isHistory bool) {
	// Select color based on role
	color := ColorGreen
	if role == openai.RoleAssistant {
		color = ColorBlue
	}

	// Apply dim intensity for historical messages
	dim := ""
	if isHistory {
		dim = ColorDim
	}

	fmt.Printf("%s%s%s%s: %s%s%s\n", dim, color, role, ColorReset, dim, message, ColorReset)
}
