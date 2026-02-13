package mcp

import (
	"sync"

	"github.com/mark3labs/mcp-go/server"
)

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
	Setup(srv *server.MCPServer) error
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
