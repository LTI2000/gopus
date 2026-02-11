package mcp

import (
	"fmt"
	"sync"
)

// Registry aggregates tools from all connected MCP servers.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool // tool name -> tool definition
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// RegisterTools adds tools from a server to the registry.
// If a tool with the same name already exists, it will be overwritten.
func (r *Registry) RegisterTools(serverID string, tools []Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, tool := range tools {
		// Set the server ID on the tool
		tool.ServerID = serverID
		r.tools[tool.Name] = tool
	}
}

// UnregisterServer removes all tools from a specific server.
func (r *Registry) UnregisterServer(serverID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, tool := range r.tools {
		if tool.ServerID == serverID {
			delete(r.tools, name)
		}
	}
}

// GetTool returns a tool by name.
func (r *Registry) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// ListTools returns all registered tools.
func (r *Registry) ListTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListToolsByServer returns all tools from a specific server.
func (r *Registry) ListToolsByServer(serverID string) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []Tool
	for _, tool := range r.tools {
		if tool.ServerID == serverID {
			tools = append(tools, tool)
		}
	}
	return tools
}

// Count returns the total number of registered tools.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// Clear removes all tools from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]Tool)
}

// ServerStats returns statistics about tools per server.
func (r *Registry) ServerStats() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]int)
	for _, tool := range r.tools {
		stats[tool.ServerID]++
	}
	return stats
}

// ValidateTool checks if a tool exists and returns an error if not.
func (r *Registry) ValidateTool(name string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.tools[name]; !ok {
		return fmt.Errorf("tool not found: %s", name)
	}
	return nil
}
