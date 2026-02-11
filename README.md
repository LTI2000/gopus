# Gopus - OpenAI Chat CLI

A command-line chat application with persistent history, automatic summarization, and MCP tool support.

## Features

- Interactive chat with conversation history
- Persistent sessions with automatic saving
- **Tiered summarization** for eternal chat history (condensed → compressed)
- **Configurable summarization prompts**
- **MCP (Model Context Protocol) support** for external tools
- Slash commands (`/summarize`, `/stats`, `/tools`, `/servers`, `/help`)
- Auto-summarization when message count exceeds threshold
- Supports all OpenAI chat models

## Quick Start

```bash
# Build (includes code generation)
make

# Configure
cp config.example.yaml config.yaml
# Edit config.yaml with your API key

# Run
./gopus
```

## Configuration

Edit `config.yaml`:

```yaml
openai:
  api_key: "sk-your-api-key"  # Required
  model: "gpt-3.5-turbo"       # Optional
  max_tokens: 1000             # Optional
  temperature: 0.7             # Optional

summarization:
  enabled: true
  recent_count: 20        # Messages kept in full
  condensed_count: 50     # Messages to condense
  auto_summarize: true
  auto_threshold: 100     # Trigger auto-summarization

# MCP (Model Context Protocol) for external tools
mcp:
  enabled: true
  tool_confirmation: "ask"  # "always", "never", or "ask"
  default_timeout: 30       # seconds
  servers:
    - name: "filesystem"
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"]
      enabled: true
```

See [`config.example.yaml`](config.example.yaml) for all options.

## Usage

### Session Management

On startup, select an existing session or start a new one:

```
=== Available Sessions ===
  0. Start a new session
  1. Explain Go interfaces (12 messages)
  2. Project planning (8 messages)

Select a session (0 for new, or number):
```

### Commands

| Command | Description |
|---------|-------------|
| `/summarize` | Manually trigger summarization |
| `/stats` | Show session statistics |
| `/tools` | List available MCP tools |
| `/servers` | Show connected MCP servers |
| `/sleep [secs]` | Test spinner animation (default: 3 seconds) |
| `/help` | List available commands |

### Controls

- `Enter` - Send message
- `Ctrl+D` - End session gracefully
- `Ctrl+C` - Immediate shutdown

## MCP (Model Context Protocol)

Gopus supports MCP for connecting to external tools. When MCP is enabled and servers are configured, the AI can use tools to:

- Read and write files
- Execute commands
- Query databases
- Access web APIs
- And more...

### Tool Confirmation

The `tool_confirmation` setting controls when you're prompted before tool execution:

| Setting | Behavior |
|---------|----------|
| `always` | Always ask before executing any tool |
| `never` | Execute tools automatically without asking |
| `ask` | Ask based on tool characteristics (default) |

### Example MCP Servers

```yaml
mcp:
  enabled: true
  servers:
    # Filesystem access
    - name: "filesystem"
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir"]
      enabled: true
    
    # GitHub integration
    - name: "github"
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-github"]
      env:
        GITHUB_TOKEN: "your-token"
      enabled: true
```

## Summarization

Gopus uses tiered summarization to maintain context over long conversations:

1. **Recent** - Last N messages kept in full detail
2. **Condensed** - Older messages summarized with key details
3. **Compressed** - Oldest messages highly compressed for long-term memory

Summarization can be triggered manually with `/summarize` or automatically when the message count exceeds the configured threshold.

## Project Structure

```
gopus/
├── main.go
├── config.example.yaml
├── Makefile
├── docs/
│   └── dependency-diagram.md  # Package dependency diagram
├── internal/
│   ├── canvas/        # Braille-based drawing canvas
│   ├── chat/          # Chat loop and commands
│   ├── config/        # Configuration loading
│   ├── history/       # Session management and storage
│   ├── mcp/           # MCP client for external tools
│   ├── openai/        # API client (oapi-codegen generated)
│   ├── printer/       # Output formatting
│   ├── signal/        # Signal handling
│   ├── spinner/       # Loading spinner (uses canvas)
│   └── summarize/     # Tiered summarization
└── plans/             # Architecture documentation
```

## Package Dependencies

See [`docs/dependency-diagram.md`](docs/dependency-diagram.md) for the full package dependency diagram.

| Package | Purpose | Key Types |
|---------|---------|-----------|
| **main** | Application entry point, orchestrates startup | - |
| **config** | YAML configuration loading with defaults | `Config`, `OpenAIConfig`, `SummarizationConfig`, `MCPConfig` |
| **openai** | OpenAI API client (generated via oapi-codegen) | `ChatClient`, `ChatCompletionRequestMessage` |
| **history** | Persistent session management with JSON storage | `Manager`, `Session`, `Message`, `Role` |
| **chat** | Interactive chat loop with slash commands | `ChatLoop` |
| **mcp** | MCP client for external tool integration | `Client`, `Registry`, `Tool`, `Transport` |
| **summarize** | Tiered message summarization (condensed → compressed) | `Summarizer`, `TierClassification`, `Stats` |
| **canvas** | Braille-based terminal drawing canvas | `Canvas` |
| **printer** | ANSI-colored terminal output | `PrintMessage()`, `PrintError()` |
| **spinner** | Animated loading indicator (uses canvas) | `Spinner` |
| **signal** | OS signal handling for graceful shutdown | `RunWithContext()` |

## Development

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make` / `make all` | Generate code and build the binary |
| `make generate` | Regenerate OpenAI client from OpenAPI spec |
| `make build` | Build the gopus binary |
| `make clean` | Remove binary and generated files |
| `make run` | Clean, build, and run the application |

### Regenerating OpenAI Client

After modifying [`internal/openai/openapi.yaml`](internal/openai/openapi.yaml):

```bash
make generate
```

## License

MIT License
