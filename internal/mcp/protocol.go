package mcp

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

// JSON-RPC 2.0 protocol constants
const (
	JSONRPCVersion = "2.0"
)

// MCP protocol version
const (
	MCPProtocolVersion = "2024-11-05"
)

// MCP method names
const (
	MethodInitialize       = "initialize"
	MethodInitialized      = "notifications/initialized"
	MethodToolsList        = "tools/list"
	MethodToolsCall        = "tools/call"
	MethodResourcesList    = "resources/list"
	MethodResourcesRead    = "resources/read"
	MethodPromptsList      = "prompts/list"
	MethodPromptsGet       = "prompts/get"
	MethodPing             = "ping"
	MethodCancelled        = "notifications/cancelled"
	MethodProgress         = "notifications/progress"
	MethodToolsListChanged = "notifications/tools/list_changed"
)

// idGenerator provides unique IDs for JSON-RPC requests.
var idGenerator atomic.Int64

// nextID returns the next unique request ID.
func nextID() int64 {
	return idGenerator.Add(1)
}

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// NewRequest creates a new JSON-RPC request with the given method and params.
func NewRequest(method string, params any) (*Request, error) {
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = data
	}

	return &Request{
		JSONRPC: JSONRPCVersion,
		ID:      nextID(),
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// Notification represents a JSON-RPC 2.0 notification (no ID, no response expected).
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// NewNotification creates a new JSON-RPC notification.
func NewNotification(method string, params any) (*Notification, error) {
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = data
	}

	return &Notification{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC error codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// InitializeParams contains the parameters for the initialize request.
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

// InitializeResult contains the result of the initialize request.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ToolsListResult contains the result of the tools/list request.
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolsCallParams contains the parameters for the tools/call request.
type ToolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolsCallResult contains the result of the tools/call request.
type ToolsCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ResourcesListResult contains the result of the resources/list request.
type ResourcesListResult struct {
	Resources []Resource `json:"resources"`
}

// ResourcesReadParams contains the parameters for the resources/read request.
type ResourcesReadParams struct {
	URI string `json:"uri"`
}

// ResourcesReadResult contains the result of the resources/read request.
type ResourcesReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

// PingResult is an empty result for the ping request.
type PingResult struct{}

// Message is a union type that can be a Request, Notification, or Response.
// Used for parsing incoming messages.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`     // nil for notifications
	Method  string          `json:"method,omitempty"` // empty for responses
	Params  json.RawMessage `json:"params,omitempty"` // for requests/notifications
	Result  json.RawMessage `json:"result,omitempty"` // for successful responses
	Error   *RPCError       `json:"error,omitempty"`  // for error responses
}

// IsRequest returns true if this message is a request (has ID and method).
func (m *Message) IsRequest() bool {
	return m.ID != nil && m.Method != ""
}

// IsNotification returns true if this message is a notification (no ID, has method).
func (m *Message) IsNotification() bool {
	return m.ID == nil && m.Method != ""
}

// IsResponse returns true if this message is a response (has ID, no method).
func (m *Message) IsResponse() bool {
	return m.ID != nil && m.Method == ""
}

// ToResponse converts the message to a Response if it is one.
func (m *Message) ToResponse() *Response {
	if !m.IsResponse() {
		return nil
	}
	return &Response{
		JSONRPC: m.JSONRPC,
		ID:      *m.ID,
		Result:  m.Result,
		Error:   m.Error,
	}
}
