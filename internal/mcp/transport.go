package mcp

import (
	"context"
	"encoding/json"
)

// Transport defines the interface for MCP communication.
type Transport interface {
	// Start begins the transport connection.
	Start(ctx context.Context) error

	// Close shuts down the transport.
	Close() error

	// Send sends a JSON-RPC message.
	Send(ctx context.Context, msg any) error

	// Receive returns a channel for receiving messages.
	Receive() <-chan *Message

	// Errors returns a channel for transport errors.
	Errors() <-chan error
}

// TransportConfig contains common transport configuration.
type TransportConfig struct {
	// Name is a human-readable name for this transport.
	Name string

	// ReadTimeout is the timeout for reading messages (0 = no timeout).
	ReadTimeout int

	// WriteTimeout is the timeout for writing messages (0 = no timeout).
	WriteTimeout int
}

// parseMessage parses a JSON message into a Message struct.
func parseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
