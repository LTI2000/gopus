# Ask Mode Rules

## Architecture Overview

- Entry point: [`main.go`](../../main.go:1) → signal handling → config → client → history → chat loop
- All packages in `internal/` are private to this module
- OpenAI client is code-generated from OpenAPI spec

## Key Packages

| Package | Purpose |
|---------|---------|
| `internal/chat` | Main chat loop, commands, spinner |
| `internal/history` | Session management, message types |
| `internal/openai` | Generated API client |
| `internal/summarize` | Auto-summarization logic |
| `internal/canvas` | Braille terminal graphics |
| `internal/animator` | Animation timing framework |

## Non-Obvious Design Decisions

- Sessions use UUID filenames, not human-readable names
- Summaries are stored as special message types within the same session file
- Braille canvas uses 2x4 pixel cells per Unicode character
