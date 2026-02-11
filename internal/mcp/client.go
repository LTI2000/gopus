package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Client manages connections to MCP servers and provides tool execution.
type Client struct {
	mu       sync.RWMutex
	servers  map[string]*ServerConnection
	registry *Registry
	config   ClientConfig
}

// ClientConfig contains configuration for the MCP client.
type ClientConfig struct {
	// ClientInfo identifies this client to servers.
	ClientInfo ClientInfo

	// DefaultTimeout is the default timeout for requests.
	DefaultTimeout time.Duration
}

// DefaultClientConfig returns a default client configuration.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		ClientInfo: ClientInfo{
			Name:    "gopus",
			Version: "1.0.0",
		},
		DefaultTimeout: 30 * time.Second,
	}
}

// ServerConnection represents a connection to a single MCP server.
type ServerConnection struct {
	ID           string
	Transport    Transport
	ServerInfo   ServerInfo
	Capabilities ServerCapabilities
	State        ConnectionState
	LastError    error

	mu             sync.Mutex
	pendingReqs    map[int64]chan *Response
	defaultTimeout time.Duration
}

// NewClient creates a new MCP client.
func NewClient(config ClientConfig) *Client {
	return &Client{
		servers:  make(map[string]*ServerConnection),
		registry: NewRegistry(),
		config:   config,
	}
}

// Registry returns the tool registry.
func (c *Client) Registry() *Registry {
	return c.registry
}

// AddServer adds and connects to an MCP server.
func (c *Client) AddServer(ctx context.Context, id string, transport Transport) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if server already exists
	if _, exists := c.servers[id]; exists {
		return fmt.Errorf("server %s already exists", id)
	}

	// Create server connection
	conn := &ServerConnection{
		ID:             id,
		Transport:      transport,
		State:          StateConnecting,
		pendingReqs:    make(map[int64]chan *Response),
		defaultTimeout: c.config.DefaultTimeout,
	}

	// Start the transport
	if err := transport.Start(ctx); err != nil {
		conn.State = StateError
		conn.LastError = err
		return fmt.Errorf("failed to start transport for %s: %w", id, err)
	}

	// Start message handler
	go c.handleMessages(conn)

	// Initialize the connection
	if err := c.initializeServer(ctx, conn); err != nil {
		conn.State = StateError
		conn.LastError = err
		transport.Close()
		return fmt.Errorf("failed to initialize server %s: %w", id, err)
	}

	// Fetch tools
	if err := c.fetchTools(ctx, conn); err != nil {
		// Non-fatal: server might not support tools
		conn.LastError = err
	}

	c.servers[id] = conn
	return nil
}

// RemoveServer disconnects and removes an MCP server.
func (c *Client) RemoveServer(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, exists := c.servers[id]
	if !exists {
		return fmt.Errorf("server %s not found", id)
	}

	// Unregister tools from this server
	c.registry.UnregisterServer(id)

	// Close the transport
	if err := conn.Transport.Close(); err != nil {
		return fmt.Errorf("failed to close transport for %s: %w", id, err)
	}

	delete(c.servers, id)
	return nil
}

// GetServer returns a server connection by ID.
func (c *Client) GetServer(id string) (*ServerConnection, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conn, exists := c.servers[id]
	return conn, exists
}

// ListServers returns all connected servers.
func (c *Client) ListServers() []*ServerConnection {
	c.mu.RLock()
	defer c.mu.RUnlock()

	servers := make([]*ServerConnection, 0, len(c.servers))
	for _, conn := range c.servers {
		servers = append(servers, conn)
	}
	return servers
}

// CallTool executes a tool and returns the result.
func (c *Client) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*ToolResult, error) {
	// Find the tool
	tool, ok := c.registry.GetTool(name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Get the server connection
	c.mu.RLock()
	conn, exists := c.servers[tool.ServerID]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server %s not connected", tool.ServerID)
	}

	// Create the request
	params := ToolsCallParams{
		Name:      name,
		Arguments: arguments,
	}

	// Send the request
	resp, err := conn.sendRequest(ctx, MethodToolsCall, params)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", name, err)
	}

	// Parse the result
	var result ToolsCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	return &ToolResult{
		ToolCallID: "", // Will be set by caller
		Content:    result.Content,
		IsError:    result.IsError,
	}, nil
}

// Close closes all server connections.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error
	for id, conn := range c.servers {
		if err := conn.Transport.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %s: %w", id, err))
		}
	}

	c.servers = make(map[string]*ServerConnection)
	c.registry.Clear()

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// initializeServer performs the MCP initialization handshake.
func (c *Client) initializeServer(ctx context.Context, conn *ServerConnection) error {
	params := InitializeParams{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{
				ListChanged: false,
			},
		},
		ClientInfo: c.config.ClientInfo,
	}

	resp, err := conn.sendRequest(ctx, MethodInitialize, params)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	conn.ServerInfo = result.ServerInfo
	conn.Capabilities = result.Capabilities
	conn.State = StateConnected

	// Send initialized notification
	notification, err := NewNotification(MethodInitialized, nil)
	if err != nil {
		return fmt.Errorf("failed to create initialized notification: %w", err)
	}

	if err := conn.Transport.Send(ctx, notification); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return nil
}

// fetchTools fetches the list of tools from a server.
func (c *Client) fetchTools(ctx context.Context, conn *ServerConnection) error {
	// Check if server supports tools
	if conn.Capabilities.Tools == nil {
		return nil
	}

	resp, err := conn.sendRequest(ctx, MethodToolsList, nil)
	if err != nil {
		return fmt.Errorf("tools/list request failed: %w", err)
	}

	var result ToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse tools list: %w", err)
	}

	// Register tools
	c.registry.RegisterTools(conn.ID, result.Tools)

	return nil
}

// handleMessages processes incoming messages from a server.
func (c *Client) handleMessages(conn *ServerConnection) {
	for msg := range conn.Transport.Receive() {
		if msg.IsResponse() {
			// Handle response to a pending request
			conn.handleResponse(msg.ToResponse())
		} else if msg.IsNotification() {
			// Handle notifications
			c.handleNotification(conn, msg)
		} else if msg.IsRequest() {
			// Handle server-initiated requests (rare)
			c.handleServerRequest(conn, msg)
		}
	}
}

// handleNotification processes a notification from a server.
func (c *Client) handleNotification(conn *ServerConnection, msg *Message) {
	switch msg.Method {
	case MethodToolsListChanged:
		// Refresh tools list
		ctx, cancel := context.WithTimeout(context.Background(), conn.defaultTimeout)
		defer cancel()
		c.fetchTools(ctx, conn)
	case MethodProgress:
		// Handle progress notifications (could be logged or displayed)
	default:
		// Unknown notification, ignore
	}
}

// handleServerRequest processes a request from a server.
func (c *Client) handleServerRequest(conn *ServerConnection, msg *Message) {
	// Most MCP servers don't send requests to clients
	// This is a placeholder for future functionality
}

// sendRequest sends a request and waits for a response.
func (conn *ServerConnection) sendRequest(ctx context.Context, method string, params any) (*Response, error) {
	req, err := NewRequest(method, params)
	if err != nil {
		return nil, err
	}

	// Create response channel
	respChan := make(chan *Response, 1)

	conn.mu.Lock()
	conn.pendingReqs[req.ID] = respChan
	conn.mu.Unlock()

	defer func() {
		conn.mu.Lock()
		delete(conn.pendingReqs, req.ID)
		conn.mu.Unlock()
	}()

	// Send the request
	if err := conn.Transport.Send(ctx, req); err != nil {
		return nil, err
	}

	// Wait for response with timeout
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// handleResponse routes a response to the waiting request.
func (conn *ServerConnection) handleResponse(resp *Response) {
	conn.mu.Lock()
	respChan, exists := conn.pendingReqs[resp.ID]
	conn.mu.Unlock()

	if exists {
		select {
		case respChan <- resp:
		default:
			// Channel full or closed, drop response
		}
	}
}
