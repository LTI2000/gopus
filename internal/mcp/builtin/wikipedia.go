package builtin

import (
	"context"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gopus/internal/mcp"
	"gopus/internal/openai"
)

func init() {
	// Register the wikipedia server with the default registry
	mcp.DefaultRegistry.Register(&WikipediaServer{})
}

// WikipediaServer is a builtin MCP server for Wikipedia searches.
type WikipediaServer struct{}

// Name returns the unique identifier for this server.
func (s *WikipediaServer) Name() string {
	return "wikipedia"
}

// Description returns a human-readable description of the server.
func (s *WikipediaServer) Description() string {
	return "Wikipedia search and summary tools"
}

// Setup configures the MCP server with tools.
// The openaiClient parameter provides access to the OpenAI API (may be nil).
func (s *WikipediaServer) Setup(srv *server.MCPServer, openaiClient *openai.ChatClient) error {
	// Add wikipedia tool - search wikipedia for a topic and return a summary
	// This tool demonstrates how to use the openaiClient in a tool handler
	srv.AddTool(mcplib.NewTool("search_wikipedia",
		mcplib.WithDescription("Search Wikipedia for a topic and return a summary"),
		mcplib.WithString("query",
			mcplib.Required(),
			mcplib.Description("The search query"),
		),
	), wikipediaToolHandler(openaiClient))

	return nil
}

// wikipediaToolHandler returns a tool handler function that has access to the OpenAI client.
// This pattern allows tools to use the OpenAI API while maintaining the required handler signature.
func wikipediaToolHandler(openaiClient *openai.ChatClient) func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		query, err := GetRequiredStringArg(req, "query")
		if err != nil {
			return nil, err
		}

		// Note: openaiClient is available here for enhanced functionality.
		// For example, you could use it to generate better summaries.
		// For now, we just demonstrate the pattern with a simulated response.
		_ = openaiClient

		summary := fmt.Sprintf("Summary for %s: This is a simulated Wikipedia article.", query)
		return mcplib.NewToolResultText(summary), nil
	}
}
