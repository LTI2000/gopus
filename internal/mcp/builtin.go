package mcp

import (
	"context"
	"sync"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gopus/internal/openai"
)

// ToolHandler is the function signature for MCP tool handlers.
type ToolHandler func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error)

// ToolHandlerFactory creates a tool handler with access to the OpenAI client.
// This allows tools to use the OpenAI API while being registered at init time.
type ToolHandlerFactory func(openaiClient *openai.ChatClient) ToolHandler

// ToolRegistration holds a tool definition and its handler factory.
type ToolRegistration struct {
	Tool           mcplib.Tool
	HandlerFactory ToolHandlerFactory
}

// ToolRegistry holds all available builtin tools.
// Builtin tools register themselves using init() functions.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolRegistration
}

// NewToolRegistry creates a new empty tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolRegistration),
	}
}

// Register adds a builtin tool to the registry.
// If a tool with the same name already exists, it will be replaced.
func (r *ToolRegistry) Register(tool mcplib.Tool, handlerFactory ToolHandlerFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = ToolRegistration{
		Tool:           tool,
		HandlerFactory: handlerFactory,
	}
}

// Get returns a tool registration by name.
func (r *ToolRegistry) Get(name string) (ToolRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	reg, ok := r.tools[name]
	return reg, ok
}

// All returns all registered tool registrations.
func (r *ToolRegistry) All() []ToolRegistration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	regs := make([]ToolRegistration, 0, len(r.tools))
	for _, reg := range r.tools {
		regs = append(regs, reg)
	}
	return regs
}

// Names returns the names of all registered tools.
func (r *ToolRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered tools.
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// DefaultToolRegistry is the global tool registry instance.
// Builtin tools should register themselves here using init() functions.
var DefaultToolRegistry = NewToolRegistry()

// BuiltinServer is the single in-process MCP server that hosts all builtin tools.
// It implements the setup logic to add all tools from DefaultToolRegistry to an MCP server.
type BuiltinServer struct{}

// Name returns the unique identifier for this server.
func (s *BuiltinServer) Name() string {
	return "builtin"
}

// Description returns a human-readable description of the server.
func (s *BuiltinServer) Description() string {
	return "Built-in MCP server hosting all registered builtin tools"
}

// Setup configures the MCP server with all tools from DefaultToolRegistry.
// The openaiClient parameter provides access to the OpenAI API for tools that need it
// (may be nil if no OpenAI client is configured).
func (s *BuiltinServer) Setup(srv *server.MCPServer, openaiClient *openai.ChatClient) error {
	// Add all tools from the DefaultToolRegistry
	for _, reg := range DefaultToolRegistry.All() {
		handler := reg.HandlerFactory(openaiClient)
		srv.AddTool(reg.Tool, func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			return handler(ctx, req)
		})
	}
	return nil
}
