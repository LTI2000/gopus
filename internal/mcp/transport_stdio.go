package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// StdioTransport implements Transport using stdio communication with a subprocess.
type StdioTransport struct {
	config StdioConfig
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	messages chan *Message
	errors   chan error

	mu       sync.Mutex
	started  bool
	closed   bool
	closeErr error
}

// StdioConfig contains configuration for stdio transport.
type StdioConfig struct {
	// Command is the command to execute.
	Command string

	// Args are the command arguments.
	Args []string

	// Env are additional environment variables (key=value format).
	Env []string

	// WorkDir is the working directory for the command.
	WorkDir string
}

// NewStdioTransport creates a new stdio transport with the given configuration.
func NewStdioTransport(config StdioConfig) *StdioTransport {
	return &StdioTransport{
		config:   config,
		messages: make(chan *Message, 100),
		errors:   make(chan error, 10),
	}
}

// Start begins the transport by starting the subprocess.
func (t *StdioTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return fmt.Errorf("transport already started")
	}

	// Create the command
	t.cmd = exec.CommandContext(ctx, t.config.Command, t.config.Args...)

	// Set up environment
	t.cmd.Env = append(os.Environ(), t.config.Env...)

	// Set working directory if specified
	if t.config.WorkDir != "" {
		t.cmd.Dir = t.config.WorkDir
	}

	// Get stdin pipe
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	t.stdin = stdin

	// Get stdout pipe
	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	t.stdout = stdout

	// Get stderr pipe for logging
	stderr, err := t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}
	t.stderr = stderr

	// Start the process
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	t.started = true

	// Start reading stdout in a goroutine
	go t.readLoop()

	// Start reading stderr in a goroutine (for logging/debugging)
	go t.readStderr()

	// Wait for process to exit in a goroutine
	go t.waitLoop()

	return nil
}

// Close shuts down the transport.
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return t.closeErr
	}
	t.closed = true

	var errs []error

	// Close stdin to signal the process to exit
	if t.stdin != nil {
		if err := t.stdin.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close stdin: %w", err))
		}
	}

	// Kill the process if it's still running
	if t.cmd != nil && t.cmd.Process != nil {
		if err := t.cmd.Process.Kill(); err != nil {
			// Ignore "process already finished" errors
			if err.Error() != "os: process already finished" {
				errs = append(errs, fmt.Errorf("failed to kill process: %w", err))
			}
		}
	}

	// Close channels
	close(t.messages)
	close(t.errors)

	if len(errs) > 0 {
		t.closeErr = errs[0]
		return t.closeErr
	}

	return nil
}

// Send sends a JSON-RPC message to the subprocess.
func (t *StdioTransport) Send(ctx context.Context, msg any) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("transport is closed")
	}
	if !t.started {
		t.mu.Unlock()
		return fmt.Errorf("transport not started")
	}
	t.mu.Unlock()

	// Marshal the message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Write the message followed by a newline
	line := append(data, '\n')

	// Check context before writing
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	_, err = t.stdin.Write(line)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Receive returns a channel for receiving messages.
func (t *StdioTransport) Receive() <-chan *Message {
	return t.messages
}

// Errors returns a channel for transport errors.
func (t *StdioTransport) Errors() <-chan error {
	return t.errors
}

// readLoop reads messages from stdout.
func (t *StdioTransport) readLoop() {
	scanner := bufio.NewScanner(t.stdout)

	// Increase buffer size for large messages
	const maxScanTokenSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		msg, err := parseMessage(line)
		if err != nil {
			t.sendError(fmt.Errorf("failed to parse message: %w", err))
			continue
		}

		// Send message to channel (non-blocking with select)
		select {
		case t.messages <- msg:
		default:
			t.sendError(fmt.Errorf("message channel full, dropping message"))
		}
	}

	if err := scanner.Err(); err != nil {
		t.sendError(fmt.Errorf("scanner error: %w", err))
	}
}

// readStderr reads and logs stderr output.
func (t *StdioTransport) readStderr() {
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		// Log stderr output for debugging
		// In production, this could be sent to a logger
		line := scanner.Text()
		if line != "" {
			// For now, we'll send stderr as errors
			// This could be changed to use a proper logger
			t.sendError(fmt.Errorf("server stderr: %s", line))
		}
	}
}

// waitLoop waits for the process to exit.
func (t *StdioTransport) waitLoop() {
	if t.cmd == nil {
		return
	}

	err := t.cmd.Wait()
	if err != nil {
		t.sendError(fmt.Errorf("process exited with error: %w", err))
	}
}

// sendError sends an error to the errors channel (non-blocking).
func (t *StdioTransport) sendError(err error) {
	select {
	case t.errors <- err:
	default:
		// Error channel full, drop the error
	}
}
