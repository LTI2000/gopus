# Gopus - OpenAI Chat CLI

A command-line chat application with persistent history and automatic summarization.

## Features

- Interactive chat with conversation history
- Persistent sessions with automatic saving
- **Tiered summarization** for eternal chat history (condensed → compressed)
- **Configurable summarization prompts**
- Slash commands (`/summarize`, `/stats`, `/help`)
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
  # condensed_prompt: |   # Custom prompt for condensed summaries
  # compressed_prompt: |  # Custom prompt for compressed summaries
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
| `/help` | List available commands |

### Controls

- `Enter` - Send message
- `Ctrl+D` - End session gracefully
- `Ctrl+C` - Immediate shutdown

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
│   ├── chat/          # Chat loop and commands
│   ├── config/        # Configuration loading
│   ├── history/       # Session management and storage
│   ├── openai/        # API client (oapi-codegen generated)
│   ├── printer/       # Output formatting
│   ├── signal/        # Signal handling
│   ├── spinner/       # Loading spinner
│   └── summarize/     # Tiered summarization
└── plans/             # Architecture documentation
```

## Package Dependencies

See [`docs/dependency-diagram.md`](docs/dependency-diagram.md) for the full package dependency diagram.

| Package | Purpose | Key Types |
|---------|---------|-----------|
| **main** | Application entry point, orchestrates startup | - |
| **config** | YAML configuration loading with defaults | `Config`, `OpenAIConfig`, `SummarizationConfig` |
| **openai** | OpenAI API client (generated via oapi-codegen) | `ChatClient`, `ChatCompletionRequestMessage` |
| **history** | Persistent session management with JSON storage | `Manager`, `Session`, `Message`, `Role` |
| **chat** | Interactive chat loop with slash commands | `ChatLoop` |
| **summarize** | Tiered message summarization (condensed → compressed) | `Summarizer`, `TierClassification`, `Stats` |
| **printer** | ANSI-colored terminal output | `PrintMessage()`, `PrintError()` |
| **spinner** | Animated loading indicator | `Spinner` |
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
