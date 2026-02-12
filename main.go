// Package main provides a simple CLI chat application using the OpenAI API.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

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

	// Create and run the chat loop
	chatLoop := chat.NewChatLoop(client, historyManager, cfg)

	// Initialize MCP client if enabled
	if cfg.MCP.Enabled {
		mcpClient, err := initMCPClient(ctx, cfg.MCP)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize MCP client: %v\n", err)
			// Continue without MCP support
		} else {
			chatLoop.SetMCPClient(mcpClient)
			defer mcpClient.Close()
		}
	}

	chatLoop.Run(ctx, scanner)

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}

// initMCPClient creates and initializes the MCP client with configured servers.
func initMCPClient(ctx context.Context, mcpCfg config.MCPConfig) (*mcp.Client, error) {
	// Create MCP client config
	clientConfig := mcp.DefaultClientConfig()
	if mcpCfg.DefaultTimeout > 0 {
		clientConfig.DefaultTimeout = time.Duration(mcpCfg.DefaultTimeout) * time.Second
	}

	// Create the MCP client
	mcpClient := mcp.NewClient(clientConfig)

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

		// Create stdio transport for this server
		transport := mcp.NewStdioTransport(mcp.StdioConfig{
			Command: serverCfg.Command,
			Args:    serverCfg.Args,
			Env:     envSlice,
			WorkDir: serverCfg.WorkDir,
		})

		// Add the server
		if err := mcpClient.AddServer(ctx, serverCfg.Name, transport); err != nil {
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
			connectedServers, len(mcpClient.Registry().ListTools()))
	}

	return mcpClient, nil
}
