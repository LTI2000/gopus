# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Project Overview

Go CLI chat application using OpenAI API with persistent session history and auto-summarization.

## Commands

```bash
make generate    # Run oapi-codegen to regenerate OpenAI client/models from openapi.yaml
make build       # Build the gopus binary
make run         # Clean, generate, build, and run
go test ./...    # Run all tests
go test ./internal/canvas/...  # Run single package tests
```

## Non-Obvious Patterns

- **Code generation**: OpenAI client is generated via `go generate` from [`internal/openai/openapi.yaml`](internal/openai/openapi.yaml:1). Config files are [`oapi-codegen-client.yaml`](internal/openai/oapi-codegen-client.yaml:1) and [`oapi-codegen-models.yaml`](internal/openai/oapi-codegen-models.yaml:1). Run `make generate` after modifying the OpenAPI spec.
- **Config file required**: App requires `config.yaml` in working directory (copy from [`config.example.yaml`](config.example.yaml:1)). Config is loaded via [`config.LoadDefault()`](internal/config/config.go:108).
- **Session storage**: Chat sessions stored in `.gopus/sessions/` directory (relative to cwd, not home). Uses JSON files with UUID filenames.
- **Spinner pattern**: Long-running operations use [`WithSpinner()`](internal/chat/spinner.go:1) wrapper for animated feedback.
- **Message types**: History distinguishes between regular messages and summaries via [`MessageType`](internal/history/message.go:19) field.
- **Braille canvas**: [`internal/canvas`](internal/canvas/canvas.go:1) uses Unicode braille characters for terminal graphics (2x4 pixel cells per character).

## Code Style

- Package comments on first file only (e.g., `// Package openai provides...`)
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Exported types have doc comments
- Standard Go project layout: `internal/` for private packages, `main.go` as entry point
