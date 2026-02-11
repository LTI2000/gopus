# Architect Mode Rules

## Architecture Constraints

- **Code generation**: OpenAI client types are generated - changes require modifying [`openapi.yaml`](../../internal/openai/openapi.yaml:1) then `make generate`
- **Session storage**: JSON files in `.gopus/sessions/` with UUID names - no database
- **Config required**: App requires `config.yaml` at runtime (not embedded)

## Component Dependencies

```
main.go
  ├── signal (context handling)
  ├── config (yaml loading)
  ├── openai (generated client)
  ├── history (session management)
  └── chat (main loop)
        ├── summarize (auto-summarization)
        ├── printer (terminal output)
        └── animator/canvas (visual feedback)
```

## Extension Points

- [`Animation`](../../internal/animator/animator.go:14) interface for custom loading animations
- [`SummarizationConfig`](../../internal/config/config.go:25) for tuning auto-summarization thresholds
- OpenAPI spec can be extended for additional API endpoints
