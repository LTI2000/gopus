// Package mcp provides a manager for MCP (Model Context Protocol) server connections.
// It wraps the github.com/mark3labs/mcp-go library to provide a unified interface
// for managing multiple MCP servers and their tools.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gopus/internal/openai"
)

// DebugTransport wraps a transport.Interface to log JSON-RPC messages.
type DebugTransport struct {
	inner    transport.Interface
	serverID string
}

// NewDebugTransport creates a new debug transport wrapper.
func NewDebugTransport(inner transport.Interface, serverID string) *DebugTransport {
	return &DebugTransport{
		inner:    inner,
		serverID: serverID,
	}
}

// Start starts the underlying transport.
func (d *DebugTransport) Start(ctx context.Context) error {
	return d.inner.Start(ctx)
}

// Close closes the underlying transport.
func (d *DebugTransport) Close() error {
	return d.inner.Close()
}

// GetSessionId returns the session ID from the underlying transport.
func (d *DebugTransport) GetSessionId() string {
	return d.inner.GetSessionId()
}

// SetNotificationHandler sets the notification handler on the underlying transport.
func (d *DebugTransport) SetNotificationHandler(handler func(notification mcplib.JSONRPCNotification)) {
	// Wrap the handler to log notifications
	d.inner.SetNotificationHandler(func(notification mcplib.JSONRPCNotification) {
		if data, err := json.Marshal(notification); err == nil {
			fmt.Fprintf(os.Stderr, "[MCP:%s] <- NOTIFICATION: %s\n", d.serverID, string(data))
		}
		if handler != nil {
			handler(notification)
		}
	})
}

// SendRequest sends a request and logs it along with the response.
func (d *DebugTransport) SendRequest(ctx context.Context, request transport.JSONRPCRequest) (*transport.JSONRPCResponse, error) {
	if data, err := json.Marshal(request); err == nil {
		fmt.Fprintf(os.Stderr, "[MCP:%s] -> REQUEST: %s\n", d.serverID, string(data))
	}

	resp, err := d.inner.SendRequest(ctx, request)

	if resp != nil {
		if data, err := json.Marshal(resp); err == nil {
			fmt.Fprintf(os.Stderr, "[MCP:%s] <- RESPONSE: %s\n", d.serverID, string(data))
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[MCP:%s] <- ERROR: %v\n", d.serverID, err)
	}

	return resp, err
}

// SendNotification sends a notification and logs it.
func (d *DebugTransport) SendNotification(ctx context.Context, notification mcplib.JSONRPCNotification) error {
	if data, err := json.Marshal(notification); err == nil {
		fmt.Fprintf(os.Stderr, "[MCP:%s] -> NOTIFICATION: %s\n", d.serverID, string(data))
	}
	return d.inner.SendNotification(ctx, notification)
}

// ToolInfo contains tool metadata with server association.
type ToolInfo struct {
	Tool     mcplib.Tool
	ServerID string
	Client   *client.Client
}

// Manager manages multiple MCP server connections.
type Manager struct {
	mu             sync.RWMutex
	clients        map[string]*client.Client
	tools          map[string]ToolInfo          // tool name -> tool info
	debug          bool                         // Enable debug logging for JSON-RPC messages
	builtinServers map[string]*server.MCPServer // Track in-process servers for cleanup
}

// NewManager creates a new MCP manager.
func NewManager() *Manager {
	return &Manager{
		clients:        make(map[string]*client.Client),
		tools:          make(map[string]ToolInfo),
		builtinServers: make(map[string]*server.MCPServer),
	}
}

// NewManagerWithDebug creates a new MCP manager with debug logging enabled.
func NewManagerWithDebug(debug bool) *Manager {
	return &Manager{
		clients:        make(map[string]*client.Client),
		tools:          make(map[string]ToolInfo),
		builtinServers: make(map[string]*server.MCPServer),
		debug:          debug,
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

	// Create the stdio client with optional debug logging
	var c *client.Client
	var err error

	if m.debug {
		// Create stdio transport, start it, and wrap it with debug logging
		stdioTransport := transport.NewStdio(command, env, args...)
		if err := stdioTransport.Start(ctx); err != nil {
			return fmt.Errorf("failed to start stdio transport for %s: %w", id, err)
		}
		debugTransport := NewDebugTransport(stdioTransport, id)
		c = client.NewClient(debugTransport)
	} else {
		c, err = client.NewStdioMCPClient(command, env, args...)
		if err != nil {
			return fmt.Errorf("failed to create client for %s: %w", id, err)
		}
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

// AddBuiltinServer registers an in-process MCP server.
// Unlike AddServer which connects to external processes via stdio,
// this method creates an in-process server that runs within the gopus process.
// The openaiClient parameter provides access to the OpenAI API for tools that need it
// (may be nil if no OpenAI client is configured).
func (m *Manager) AddBuiltinServer(ctx context.Context, builtin *BuiltinServer, openaiClient *openai.ChatClient) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := builtin.Name()

	// Check if server already exists
	if _, exists := m.clients[id]; exists {
		return fmt.Errorf("server %s already exists", id)
	}

	// Create the MCP server
	srv := server.NewMCPServer(
		id,
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Let the builtin server configure itself (add tools, resources, etc.)
	if err := builtin.Setup(srv, openaiClient); err != nil {
		return fmt.Errorf("failed to setup builtin server %s: %w", id, err)
	}

	// Create in-process transport
	inProcessTransport := transport.NewInProcessTransport(srv)
	if err := inProcessTransport.Start(ctx); err != nil {
		return fmt.Errorf("failed to start in-process transport for %s: %w", id, err)
	}

	// Optionally wrap with debug transport
	var c *client.Client
	if m.debug {
		debugTransport := NewDebugTransport(inProcessTransport, id)
		c = client.NewClient(debugTransport)
	} else {
		c = client.NewClient(inProcessTransport)
	}

	// Initialize the client
	initRequest := mcplib.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcplib.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcplib.Implementation{
		Name:    "gopus",
		Version: "1.0.0",
	}

	_, err := c.Initialize(ctx, initRequest)
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize builtin server %s: %w", id, err)
	}

	// Store the client and server
	m.clients[id] = c
	m.builtinServers[id] = srv

	// Fetch and register tools
	if err := m.fetchTools(ctx, id, c); err != nil {
		// Non-fatal: server might not have tools yet
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
	m.builtinServers = make(map[string]*server.MCPServer)

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
