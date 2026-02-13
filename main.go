// Package main provides a simple CLI chat application using the OpenAI API.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"gopus/internal/chat"
	"gopus/internal/config"
	"gopus/internal/history"
	"gopus/internal/mcp"
	"gopus/internal/openai"
	"gopus/internal/signal"
)

func main() {
	// Set up signal handling for graceful shutdown
	signal.RunWithContext(main0)
}

func main0(ctx context.Context) {
	fmt.Printf("Press Ctrl+D to end the session.\n")

	// Create scanner for reading user input
	scanner := bufio.NewScanner(os.Stdin)

	// Load configuration
	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please copy config.example.yaml to config.yaml and add your API key.")
		os.Exit(1)
	}

	// Create OpenAI client
	client, err := openai.NewChatClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Initialize history manager (use configured sessions_dir or default)
	historyManager, err := history.NewManager(cfg.History.SessionsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing history: %v\n", err)
		os.Exit(1)
	}

	// Session selection at startup
	if err := history.SelectSession(historyManager, scanner); err != nil {
		fmt.Fprintf(os.Stderr, "Error selecting session: %v\n", err)
		os.Exit(1)
	}

	// Initialize MCP manager
	mcpManager, err := initMCPManager(ctx, cfg.MCP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize MCP manager: %v\n", err)
		// Continue without MCP support
	} else {
		defer mcpManager.Close()
	}

	// Create and run the chat loop
	chatLoop := chat.NewChatLoop(client, historyManager, mcpManager, cfg)

	chatLoop.Run(ctx, scanner)

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}

// initMCPManager creates and initializes the MCP manager with configured servers.
func initMCPManager(ctx context.Context, mcpCfg config.MCPConfig) (*mcp.Manager, error) {
	// Create the MCP manager with optional debug logging
	manager := mcp.NewManagerWithDebug(mcpCfg.Debug)

	if mcpCfg.Debug {
		fmt.Fprintln(os.Stderr, "MCP debug logging enabled - JSON-RPC messages will be displayed")
	}

	// Connect to each enabled server
	connectedServers := 0
	for _, serverCfg := range mcpCfg.Servers {
		if !serverCfg.Enabled {
			continue
		}

		// Convert env map to slice format
		var envSlice []string
		for k, v := range serverCfg.Env {
			envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
		}

		// Add the server (uses stdio transport internally)
		if err := manager.AddServer(ctx, serverCfg.Name, serverCfg.Command, envSlice, serverCfg.Args...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to connect to MCP server %q: %v\n", serverCfg.Name, err)
			continue
		}

		fmt.Printf("Connected to MCP server: %s\n", serverCfg.Name)
		connectedServers++
	}

	if connectedServers == 0 && len(mcpCfg.Servers) > 0 {
		return nil, fmt.Errorf("no MCP servers connected successfully")
	}

	if connectedServers > 0 {
		fmt.Printf("MCP: %d server(s) connected, %d tool(s) available\n",
			connectedServers, manager.ToolCount())
	}

	return manager, nil
}
