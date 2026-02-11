// Package mcp provides a client for the Model Context Protocol (MCP).
// MCP enables AI applications to connect to external tools and resources
// through a standardized JSON-RPC based protocol.
package mcp

import "encoding/json"

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
	ServerID    string          `json:"-"` // Which server provides this tool (not serialized)
}

// ToolCall represents a request to execute a tool.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolCallID string        `json:"tool_call_id"`
	Content    []ToolContent `json:"content"`
	IsError    bool          `json:"isError,omitempty"`
}

// ToolContent represents content returned from a tool.
type ToolContent struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
	// Additional fields for other content types can be added as needed
}

// Resource represents an MCP resource.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceContent represents the content of a resource.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // Base64 encoded binary data
}

// ServerInfo contains information about an MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities describes what an MCP server supports.
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability indicates the server supports tools.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates the server supports resources.
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates the server supports prompts.
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ClientInfo contains information about the MCP client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities describes what the MCP client supports.
type ClientCapabilities struct {
	Roots    *RootsCapability    `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

// RootsCapability indicates the client supports roots.
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability indicates the client supports sampling.
type SamplingCapability struct{}

// ConnectionState represents the state of an MCP server connection.
type ConnectionState int

const (
	// StateDisconnected means the server is not connected.
	StateDisconnected ConnectionState = iota
	// StateConnecting means the connection is being established.
	StateConnecting
	// StateConnected means the server is connected and ready.
	StateConnected
	// StateError means the connection is in an error state.
	StateError
)

// String returns a human-readable representation of the connection state.
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}
