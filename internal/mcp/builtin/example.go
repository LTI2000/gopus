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
func (s *ExampleServer) Setup(srv *server.MCPServer) error {
	// Add echo tool - simply echoes back the input
	echoTool := mcplib.NewTool("echo",
		mcplib.WithDescription("Echoes back the input message"),
		mcplib.WithString("message",
			mcplib.Required(),
			mcplib.Description("The message to echo back"),
		),
	)

	srv.AddTool(echoTool, func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		args, ok := req.Params.Arguments.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid arguments format")
		}
		message, ok := args["message"].(string)
		if !ok {
			return nil, fmt.Errorf("message argument is required and must be a string")
		}
		return mcplib.NewToolResultText(fmt.Sprintf("Echo: %s", message)), nil
	})

	// Add current_time tool - returns the current time
	timeTool := mcplib.NewTool("current_time",
		mcplib.WithDescription("Returns the current date and time"),
		mcplib.WithString("format",
			mcplib.Description("Time format (optional). Use 'unix' for Unix timestamp, 'iso' for ISO 8601, or a Go time format string. Default: RFC3339"),
		),
	)

	srv.AddTool(timeTool, func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		now := time.Now()
		format := "RFC3339" // default

		if args, ok := req.Params.Arguments.(map[string]any); ok {
			if f, ok := args["format"].(string); ok && f != "" {
				format = f
			}
		}

		var result string
		switch format {
		case "unix":
			result = fmt.Sprintf("%d", now.Unix())
		case "iso", "ISO8601", "RFC3339":
			result = now.Format(time.RFC3339)
		default:
			// Try to use as a Go time format string
			result = now.Format(format)
		}

		return mcplib.NewToolResultText(result), nil
	})

	return nil
}
