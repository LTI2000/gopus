// Package chat provides the main chat loop functionality.
package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopus/internal/config"
	"gopus/internal/history"
	"gopus/internal/mcp"
	"gopus/internal/openai"
	"gopus/internal/printer"
	"gopus/internal/summarize"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// ChatLoop holds the dependencies for the chat loop.
type ChatLoop struct {
	client         *openai.ChatClient
	historyManager *history.Manager
	summarizer     *summarize.Summarizer
	mcpManager     *mcp.Manager
	config         *config.Config
}

// NewChatLoop creates a new chat loop with the given dependencies.
func NewChatLoop(client *openai.ChatClient, historyManager *history.Manager, mcpManager *mcp.Manager, cfg *config.Config) *ChatLoop {
	return &ChatLoop{
		client:         client,
		historyManager: historyManager,
		summarizer:     summarize.New(client, cfg.Summarization),
		mcpManager:     mcpManager,
		config:         cfg,
	}
}

// Run runs the main chat loop, reading user input and sending requests to OpenAI.
func (c *ChatLoop) Run(ctx context.Context, scanner *bufio.Scanner) {
	// Display help at startup
	c.handleHelp()

	// Convert session messages to OpenAI format for API calls
	session := c.historyManager.Current()
	chatHistory := history.MessagesToOpenAI(session.Messages)

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

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if c.handleCommand(ctx, input, &chatHistory) {
				continue
			}
		}

		// Add user message to history manager (auto-saves)
		if err := c.historyManager.AddMessage(history.RoleUser, input); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving message: %v\n", err)
		}

		// Add user message to chat history for API
		chatHistory = append(chatHistory, openai.ChatCompletionRequestMessage{
			Role:    openai.RoleUser,
			Content: &input,
		})

		// Process the conversation (may involve multiple tool calls)
		if err := c.processConversation(ctx, &chatHistory); err != nil {
			printer.PrintError("Error: %v", err)
			// Remove the failed message from both histories
			chatHistory = chatHistory[:len(chatHistory)-1]
			// Remove from session history too
			session := c.historyManager.Current()
			if len(session.Messages) > 0 {
				session.Messages = session.Messages[:len(session.Messages)-1]
				c.historyManager.SaveCurrent()
			}
			continue
		}

		// Check for auto-summarization
		c.checkAutoSummarize(ctx, &chatHistory)
	}
}

// processConversation handles the conversation loop including tool calls.
func (c *ChatLoop) processConversation(ctx context.Context, chatHistory *[]openai.ChatCompletionRequestMessage) error {
	// Get tools from MCP client if available
	tools := c.getOpenAITools()

	for {
		// Send request to OpenAI with spinner
		resp, err := WithSpinner(func() (*openai.ChatCompletionResponse, error) {
			return c.client.ChatCompletionWithTools(ctx, *chatHistory, tools)
		})

		if err != nil {
			return err
		}

		// Extract the response
		if len(resp.Choices) == 0 {
			return fmt.Errorf("no response from API")
		}

		choice := resp.Choices[0]
		message := choice.Message

		// Check if the model wants to call tools
		if message.ToolCalls != nil && len(*message.ToolCalls) > 0 {
			// Add assistant message with tool calls to history
			assistantMsg := c.buildAssistantMessageWithToolCalls(message)
			*chatHistory = append(*chatHistory, assistantMsg)

			// Display pending tool calls
			fmt.Printf("\n%s[AI wants to call %d tool(s)]%s\n", printer.ColorYellow, len(*message.ToolCalls), printer.ColorReset)
			for _, tc := range *message.ToolCalls {
				fmt.Printf("  • %s%s%s(%s)\n", printer.ColorCyan, tc.Function.Name, printer.ColorReset, tc.Function.Arguments)
			}

			// Check confirmation setting
			if !c.confirmToolExecution(*message.ToolCalls) {
				// User declined - add a declined message and return
				declinedMsg := "Tool execution was declined by the user."
				for _, toolCall := range *message.ToolCalls {
					toolResultMsg := c.buildToolResultMessage(toolCall.Id, declinedMsg)
					*chatHistory = append(*chatHistory, toolResultMsg)
				}
				fmt.Printf("%s[Tool execution declined]%s\n", printer.ColorYellow, printer.ColorReset)
				continue
			}

			// Execute each tool call
			for _, toolCall := range *message.ToolCalls {
				fmt.Printf("%s[Executing %s...]%s\n", printer.ColorCyan, toolCall.Function.Name, printer.ColorReset)
				result, err := c.executeToolCall(ctx, toolCall)
				if err != nil {
					// Add error result to history
					toolResultMsg := c.buildToolResultMessage(toolCall.Id, fmt.Sprintf("Error: %v", err))
					*chatHistory = append(*chatHistory, toolResultMsg)
					fmt.Printf("%s[Tool %s failed: %v]%s\n", printer.ColorRed, toolCall.Function.Name, err, printer.ColorReset)
				} else {
					// Add success result to history
					toolResultMsg := c.buildToolResultMessage(toolCall.Id, result)
					*chatHistory = append(*chatHistory, toolResultMsg)
					fmt.Printf("%s[Tool %s completed]%s\n", printer.ColorGreen, toolCall.Function.Name, printer.ColorReset)
				}
			}

			// Continue the loop to get the model's response after tool execution
			continue
		}

		// No tool calls - this is the final response
		if message.Content == nil {
			return fmt.Errorf("empty response from API")
		}

		assistantMessage := *message.Content
		printer.PrintMessage(string(history.RoleAssistant), assistantMessage, false)
		fmt.Println()

		// Add assistant response to history manager (auto-saves)
		if err := c.historyManager.AddMessage(history.RoleAssistant, assistantMessage); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving message: %v\n", err)
		}

		// Add assistant response to chat history for API
		*chatHistory = append(*chatHistory, openai.ChatCompletionRequestMessage{
			Role:    openai.RoleAssistant,
			Content: &assistantMessage,
		})

		return nil
	}
}

// getOpenAITools converts MCP tools to OpenAI format.
func (c *ChatLoop) getOpenAITools() []openai.ChatCompletionTool {
	if c.mcpManager == nil {
		return nil
	}

	mcpTools := c.mcpManager.ListTools()
	if len(mcpTools) == 0 {
		return nil
	}

	tools := make([]openai.ChatCompletionTool, 0, len(mcpTools))
	for _, tool := range mcpTools {
		// Convert MCP tool schema to OpenAI format
		// Marshal the InputSchema to JSON and unmarshal to map[string]interface{}
		schemaBytes, err := json.Marshal(tool.InputSchema)
		if err != nil {
			continue // Skip tools with invalid schemas
		}

		var params map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &params); err != nil {
			continue // Skip tools with invalid schemas
		}

		tools = append(tools, openai.ChatCompletionTool{
			Type: openai.Function,
			Function: openai.FunctionDefinition{
				Name:        tool.Name,
				Description: &tool.Description,
				Parameters:  &params,
			},
		})
	}

	return tools
}

// buildAssistantMessageWithToolCalls creates an assistant message containing tool calls.
func (c *ChatLoop) buildAssistantMessageWithToolCalls(message openai.ChatCompletionResponseMessage) openai.ChatCompletionRequestMessage {
	role := openai.ChatCompletionRequestMessageRoleAssistant

	// Convert response tool calls to request format
	var toolCalls []openai.ChatCompletionMessageToolCall
	if message.ToolCalls != nil {
		for _, tc := range *message.ToolCalls {
			toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCall{
				Id:   tc.Id,
				Type: openai.ChatCompletionMessageToolCallTypeFunction,
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return openai.ChatCompletionRequestMessage{
		Role:      role,
		Content:   message.Content,
		ToolCalls: &toolCalls,
	}
}

// buildToolResultMessage creates a tool result message.
func (c *ChatLoop) buildToolResultMessage(toolCallID, content string) openai.ChatCompletionRequestMessage {
	role := openai.ChatCompletionRequestMessageRoleTool
	return openai.ChatCompletionRequestMessage{
		Role:       role,
		Content:    &content,
		ToolCallId: &toolCallID,
	}
}

// executeToolCall executes a single tool call via MCP.
func (c *ChatLoop) executeToolCall(ctx context.Context, toolCall openai.ChatCompletionMessageToolCall) (string, error) {
	if c.mcpManager == nil {
		return "", fmt.Errorf("MCP manager not configured")
	}

	// Parse the arguments into map[string]any
	var args map[string]any
	if toolCall.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	}

	// Call the tool
	result, err := c.mcpManager.CallTool(ctx, toolCall.Function.Name, args)
	if err != nil {
		return "", err
	}

	// Format the result content
	if result.IsError {
		return fmt.Sprintf("Tool error: %s", c.formatToolContent(result.Content)), nil
	}

	return c.formatToolContent(result.Content), nil
}

// formatToolContent formats tool result content for display.
func (c *ChatLoop) formatToolContent(content []mcplib.Content) string {
	var parts []string
	for _, item := range content {
		switch c := item.(type) {
		case mcplib.TextContent:
			parts = append(parts, c.Text)
		case *mcplib.TextContent:
			parts = append(parts, c.Text)
		case mcplib.ImageContent:
			parts = append(parts, "[image content]")
		case *mcplib.ImageContent:
			parts = append(parts, "[image content]")
		case mcplib.AudioContent:
			parts = append(parts, "[audio content]")
		case *mcplib.AudioContent:
			parts = append(parts, "[audio content]")
		default:
			parts = append(parts, "[unknown content]")
		}
	}
	return strings.Join(parts, "\n")
}

// confirmToolExecution checks if tool execution should proceed based on config.
// Returns true if execution should proceed, false if declined.
func (c *ChatLoop) confirmToolExecution(toolCalls []openai.ChatCompletionMessageToolCall) bool {
	confirmation := c.config.MCP.ToolConfirmation

	switch confirmation {
	case config.ToolConfirmationNever:
		// Never ask, always execute
		return true

	case config.ToolConfirmationAlways:
		// Always ask for confirmation
		return c.promptForConfirmation(toolCalls)

	case config.ToolConfirmationAsk:
		// Ask based on tool characteristics (for now, always ask)
		// In the future, this could check tool metadata for risk level
		return c.promptForConfirmation(toolCalls)

	default:
		// Unknown setting, default to asking
		return c.promptForConfirmation(toolCalls)
	}
}

// promptForConfirmation asks the user to confirm tool execution.
func (c *ChatLoop) promptForConfirmation(toolCalls []openai.ChatCompletionMessageToolCall) bool {
	fmt.Printf("\n%sExecute these tools? [y/N]: %s", printer.ColorYellow, printer.ColorReset)

	// Read a single line of input
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// checkAutoSummarize checks if auto-summarization should be triggered.
func (c *ChatLoop) checkAutoSummarize(ctx context.Context, chatHistory *[]openai.ChatCompletionRequestMessage) {
	session := c.historyManager.Current()

	if !c.summarizer.ShouldAutoSummarize(session.Messages) {
		return
	}

	fmt.Println("\n[Auto-summarizing history...]")

	// Process the session with spinner
	newMessages, err := WithSpinner(func() ([]history.Message, error) {
		return c.summarizer.ProcessSession(ctx, session)
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Auto-summarization error: %v\n", err)
		return
	}

	// Update session with summarized messages
	oldCount := len(session.Messages)
	session.Messages = newMessages
	if err := c.historyManager.SaveCurrent(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving session: %v\n", err)
		return
	}

	// Update the chat history for API calls
	*chatHistory = history.MessagesToOpenAI(newMessages)

	fmt.Printf("[✓ Auto-summarized: %d → %d messages]\n\n", oldCount, len(newMessages))
}
