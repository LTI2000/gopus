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

// BuiltinServer defines the interface for in-process MCP servers.
// Builtin servers run within the gopus process and don't require
// external processes or stdio communication.
type BuiltinServer interface {
	// Name returns the unique identifier for this server.
	// This is used as the server ID in the Manager.
	Name() string

	// Description returns a human-readable description of the server.
	Description() string

	// Setup configures the MCP server with tools, resources, prompts, etc.
	// This is called once when the server is registered with the Manager.
	// The server should add its tools using srv.AddTool() and similar methods.
	// The openaiClient parameter provides access to the OpenAI API for tools
	// that need it (may be nil if no OpenAI client is configured).
	Setup(srv *server.MCPServer, openaiClient *openai.ChatClient) error
}

// Registry holds all available builtin servers.
// Builtin servers register themselves using init() functions.
type Registry struct {
	mu      sync.RWMutex
	servers map[string]BuiltinServer
}

// NewRegistry creates a new empty registry.
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]BuiltinServer),
	}
}

// Register adds a builtin server to the registry.
// If a server with the same name already exists, it will be replaced.
func (r *Registry) Register(srv BuiltinServer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers[srv.Name()] = srv
}

// Get returns a builtin server by name.
func (r *Registry) Get(name string) (BuiltinServer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	srv, ok := r.servers[name]
	return srv, ok
}

// All returns all registered builtin servers.
func (r *Registry) All() []BuiltinServer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	servers := make([]BuiltinServer, 0, len(r.servers))
	for _, srv := range r.servers {
		servers = append(servers, srv)
	}
	return servers
}

// Names returns the names of all registered builtin servers.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.servers))
	for name := range r.servers {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered builtin servers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.servers)
}

// DefaultRegistry is the global registry instance.
// Builtin servers should register themselves here using init() functions.
var DefaultRegistry = NewRegistry()
