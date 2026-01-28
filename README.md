# Gopus - OpenAI Chat CLI

A simple command-line chat application that uses the OpenAI Chat Completions API.

## Features

- Interactive chat loop with conversation history
- Configuration file-based API key management
- Supports all OpenAI chat models (GPT-3.5, GPT-4, etc.)
- Configurable temperature and max tokens
- Graceful shutdown with Ctrl+C

## Prerequisites

- Go 1.21 or later
- An OpenAI API key

## Installation

1. Clone or download this repository:
   ```bash
   cd gopus
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build the application:
   ```bash
   go build -o gopus
   ```

## Configuration

1. Copy the example configuration file:
   ```bash
   cp config.example.yaml config.yaml
   ```

2. Edit `config.yaml` and add your OpenAI API key:
   ```yaml
   openai:
     api_key: "sk-your-actual-api-key"
     model: "gpt-3.5-turbo"
     max_tokens: 1000
     temperature: 0.7
   ```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `api_key` | Your OpenAI API key (required) | - |
| `model` | The model to use for completions | `gpt-3.5-turbo` |
| `max_tokens` | Maximum tokens in the response | `1000` |
| `temperature` | Response randomness (0.0-2.0) | `0.7` |
| `base_url` | API base URL (for proxies) | `https://api.openai.com/v1` |

## Usage

Run the application:

```bash
./gopus
```

Or run directly with Go:

```bash
go run main.go
```

### Example Session

```
Loading configuration from config.yaml...
Connected to OpenAI (model: gpt-3.5-turbo). Type 'quit' or 'exit' to end the conversation.

You: Hello! What can you help me with?
Assistant: Hello! I'm an AI assistant and I can help you with a wide variety of tasks including:
- Answering questions on many topics
- Writing and editing text
- Explaining concepts
- Helping with coding problems
- And much more!

What would you like to know?

You: Can you explain what Go interfaces are?
Assistant: Go interfaces are a powerful feature that define a set of method signatures...

You: quit
Goodbye!
```

### Commands

- Type your message and press Enter to send
- Type `quit` or `exit` to end the conversation
- Press `Ctrl+C` for immediate shutdown

## Project Structure

```
gopus/
├── main.go                 # Application entry point with chat loop
├── config.yaml             # Your configuration (not in git)
├── config.example.yaml     # Example configuration template
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── README.md               # This file
└── internal/
    ├── config/
    │   └── config.go       # Configuration loading
    └── openai/
        ├── client.go       # OpenAI API client
        └── types.go        # API request/response types
```

## Security Notes

- Never commit your `config.yaml` file with your API key
- The `config.yaml` file is included in `.gitignore`
- API keys are never logged or displayed

## License

MIT License
