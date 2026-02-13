// Package builtin provides builtin in-process MCP servers for gopus.
// Builtin servers run within the gopus process and don't require external processes.
package builtin

import (
	"context"
	"fmt"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gopus/internal/mcp"
	"gopus/internal/openai"
)

func init() {
	// Register the example server with the default registry
	mcp.DefaultRegistry.Register(&ExampleServer{})
}

// ExampleServer is a simple example builtin MCP server.
// It demonstrates how to create a builtin server with tools.
type ExampleServer struct{}

// Name returns the unique identifier for this server.
func (s *ExampleServer) Name() string {
	return "example"
}

// Description returns a human-readable description of the server.
func (s *ExampleServer) Description() string {
	return "Example builtin server demonstrating in-process MCP tools"
}

// Setup configures the MCP server with tools.
// The openaiClient parameter provides access to the OpenAI API (may be nil).
func (s *ExampleServer) Setup(srv *server.MCPServer, openaiClient *openai.ChatClient) error {
	// Add echo tool - simply echoes back the input
	srv.AddTool(
		mcplib.NewTool("echo",
			mcplib.WithDescription("Echoes back the input message"),
			mcplib.WithString("message",
				mcplib.Required(),
				mcplib.Description("The message to echo back"),
			),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			message, err := GetRequiredStringArg(req, "message")
			if err != nil {
				return nil, err
			}
			return mcplib.NewToolResultText(fmt.Sprintf("Echo: %s", message)), nil
		},
	)

	// Add current_time tool - returns the current time
	srv.AddTool(
		mcplib.NewTool("current_time",
			mcplib.WithDescription("Returns the current date and time"),
			mcplib.WithString("format",
				mcplib.Description("Time format (optional). Use 'unix' for Unix timestamp, 'iso' for ISO 8601, or a Go time format string. Default: RFC3339"),
			),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
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
		},
	)

	return nil
}
