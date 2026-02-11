# Debug Mode Rules

## Common Issues

- **Missing config.yaml**: App exits with "Please copy config.example.yaml to config.yaml"
- **Session directory**: Sessions stored in `.gopus/sessions/` relative to cwd, not home directory
- **Generated code errors**: Run `make generate` if OpenAI client types are missing or outdated

## Testing

```bash
go test ./...                      # All tests
go test ./internal/canvas/...      # Single package
go test -v ./internal/canvas/...   # Verbose output
```

## Key Files for Debugging

- [`internal/config/config.go`](../../internal/config/config.go:1) - Config loading and defaults
- [`internal/history/storage.go`](../../internal/history/storage.go:1) - Session persistence
- [`internal/openai/client.go`](../../internal/openai/client.go:1) - API client and error handling
