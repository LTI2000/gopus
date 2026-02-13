// Package builtin provides builtin in-process MCP tools for gopus.
// Builtin tools are registered with the DefaultToolRegistry and run within the gopus process.
package builtin

import (
	"context"
	"fmt"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"gopus/internal/mcp"
	"gopus/internal/openai"
)

func init() {
	// Register tools with the default tool registry
	mcp.DefaultToolRegistry.Register(
		mcplib.NewTool("echo",
			mcplib.WithDescription("Echoes back the input message"),
			mcplib.WithString("message",
				mcplib.Required(),
				mcplib.Description("The message to echo back"),
			),
		),
		func(openaiClient *openai.ChatClient) mcp.ToolHandler {
			return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
				message, err := GetRequiredStringArg(req, "message")
				if err != nil {
					return nil, err
				}
				return mcplib.NewToolResultText(fmt.Sprintf("Echo: %s", message)), nil
			}
		},
	)

	mcp.DefaultToolRegistry.Register(
		mcplib.NewTool("current_time",
			mcplib.WithDescription("Returns the current date and time"),
			mcplib.WithString("format",
				mcplib.Description("Time format (optional). Use 'unix' for Unix timestamp, 'iso' for ISO 8601, or a Go time format string. Default: RFC3339"),
			),
		),
		func(openaiClient *openai.ChatClient) mcp.ToolHandler {
			return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
				now := time.Now()

				args, _ := GetArgs(req)
				format := GetOptionalStringArg(args, "format", "RFC3339")

				var result string
				switch format {
				case "unix":
					result = fmt.Sprintf("%d", now.Unix())
				case "iso", "ISO8601", "RFC3339":
					result = now.Format(time.RFC3339)
				default:
					result = now.Format(format)
				}

				return mcplib.NewToolResultText(result), nil
			}
		},
	)
}
