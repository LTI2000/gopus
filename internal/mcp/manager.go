// Package mcp provides a manager for MCP (Model Context Protocol) server connections.
// It wraps the github.com/mark3labs/mcp-go library to provide a unified interface
// for managing multiple MCP servers and their tools.
package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// ToolInfo contains tool metadata with server association.
type ToolInfo struct {
	Tool     mcplib.Tool
	ServerID string
	Client   *client.Client
}

// Manager manages multiple MCP server connections.
type Manager struct {
	mu      sync.RWMutex
	clients map[string]*client.Client
	tools   map[string]ToolInfo // tool name -> tool info
}

// NewManager creates a new MCP manager.
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*client.Client),
		tools:   make(map[string]ToolInfo),
	}
}

// AddServer connects to an MCP server via stdio and initializes it.
func (m *Manager) AddServer(ctx context.Context, id, command string, env []string, args ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if server already exists
	if _, exists := m.clients[id]; exists {
		return fmt.Errorf("server %s already exists", id)
	}

	// Create the stdio client
	c, err := client.NewStdioMCPClient(command, env, args...)
	if err != nil {
		return fmt.Errorf("failed to create client for %s: %w", id, err)
	}

	// Initialize the client
	initRequest := mcplib.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcplib.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcplib.Implementation{
		Name:    "gopus",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initRequest)
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize server %s: %w", id, err)
	}

	// Store the client
	m.clients[id] = c

	// Fetch and register tools
	if err := m.fetchTools(ctx, id, c); err != nil {
		// Non-fatal: server might not support tools
		// Log but continue
	}

	return nil
}

// fetchTools fetches tools from a server and registers them.
func (m *Manager) fetchTools(ctx context.Context, serverID string, c *client.Client) error {
	toolsRequest := mcplib.ListToolsRequest{}
	result, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	for _, tool := range result.Tools {
		m.tools[tool.Name] = ToolInfo{
			Tool:     tool,
			ServerID: serverID,
			Client:   c,
		}
	}

	return nil
}

// RemoveServer disconnects and removes an MCP server.
func (m *Manager) RemoveServer(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, exists := m.clients[id]
	if !exists {
		return fmt.Errorf("server %s not found", id)
	}

	// Remove tools from this server
	for name, info := range m.tools {
		if info.ServerID == id {
			delete(m.tools, name)
		}
	}

	// Close the client
	if err := c.Close(); err != nil {
		return fmt.Errorf("failed to close client for %s: %w", id, err)
	}

	delete(m.clients, id)
	return nil
}

// ListTools returns all available tools from all connected servers.
func (m *Manager) ListTools() []mcplib.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]mcplib.Tool, 0, len(m.tools))
	for _, info := range m.tools {
		tools = append(tools, info.Tool)
	}
	return tools
}

// GetTool returns a tool by name.
func (m *Manager) GetTool(name string) (mcplib.Tool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.tools[name]
	if !ok {
		return mcplib.Tool{}, false
	}
	return info.Tool, true
}

// ToolCount returns the total number of registered tools.
func (m *Manager) ToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tools)
}

// ServerCount returns the number of connected servers.
func (m *Manager) ServerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// CallTool executes a tool by name with the given arguments.
func (m *Manager) CallTool(ctx context.Context, name string, arguments map[string]any) (*mcplib.CallToolResult, error) {
	m.mu.RLock()
	info, ok := m.tools[name]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Build the call request
	callRequest := mcplib.CallToolRequest{}
	callRequest.Params.Name = name
	callRequest.Params.Arguments = arguments

	result, err := info.Client.CallTool(ctx, callRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", name, err)
	}

	return result, nil
}

// Close closes all client connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for id, c := range m.clients {
		if err := c.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", id, err))
		}
	}

	m.clients = make(map[string]*client.Client)
	m.tools = make(map[string]ToolInfo)

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// RefreshTools refreshes the tool list from all connected servers.
func (m *Manager) RefreshTools(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing tools
	m.tools = make(map[string]ToolInfo)

	// Fetch tools from all servers
	var lastErr error
	for id, c := range m.clients {
		if err := m.fetchTools(ctx, id, c); err != nil {
			lastErr = err
		}
	}

	return lastErr
}
