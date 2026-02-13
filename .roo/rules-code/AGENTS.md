# Code Mode Rules

## Code Generation

- After modifying [`internal/openai/openapi.yaml`](../../internal/openai/openapi.yaml:1), run `make generate` to regenerate client/models
- Generated files: `client_gen.go` and `models_gen.go` - do NOT edit manually

## Patterns to Follow

- Use [`WithSpinner()`](../../internal/chat/spinner.go:177) wrapper for any long-running operations (API calls, file I/O)
- History messages use [`MessageType`](../../internal/history/message.go:21) to distinguish regular messages from summaries
- Config loading via [`config.LoadDefault()`](../../internal/config/config.go:178) - requires `config.yaml` in cwd

## Error Handling

- Always wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Use `printer.PrintError()` for user-facing errors in chat context
