# Property-Based Testing Design for gopus

## Overview

This document outlines a property-based testing approach for the gopus project, focusing on minimal dependencies and clean integration with Go's standard testing patterns.

## Library Recommendation: `testing/quick`

### Rationale

After evaluating the available options, **`testing/quick`** is the recommended choice for this project:

| Library | Dependencies | Features | Recommendation |
|---------|-------------|----------|----------------|
| `testing/quick` | None (stdlib) | Basic property testing, automatic value generation | ✅ **Recommended** |
| `gopter` | External | Generators, shrinking, stateful testing | Overkill for current needs |
| `rapid` | External | Modern API, better shrinking | Good alternative if stdlib insufficient |

**Key reasons for `testing/quick`:**

1. **Zero dependencies** - Aligns with project's minimal dependency philosophy
2. **Built into Go** - No version management, always available
3. **Sufficient for current codebase** - The functions identified for testing have simple input domains
4. **Familiar patterns** - Uses standard `*testing.T` integration
5. **Project already uses Go 1.25** - Full stdlib support available

### When to Consider Alternatives

Consider `rapid` if you later need:
- Custom generators for complex types
- Better shrinking for failure cases
- Stateful/sequential testing

## Candidates for Property-Based Testing

### 1. Canvas Package - [`internal/canvas/canvas.go`](../internal/canvas/canvas.go:1)

The canvas package is an **ideal candidate** for property-based testing due to its pure, mathematical nature.

#### Properties to Test

| Function | Property | Description |
|----------|----------|-------------|
| [`New()`](../internal/canvas/canvas.go:40) | Dimension rounding | Width always even, height always multiple of 4 |
| [`New()`](../internal/canvas/canvas.go:40) | Dimension growth | Output dimensions >= input dimensions |
| [`Set()`](../internal/canvas/canvas.go:87) / [`Get()`](../internal/canvas/canvas.go:109) | Round-trip | `Set(x,y)` then `Get(x,y)` returns true |
| [`Toggle()`](../internal/canvas/canvas.go:101) | Involution | Double toggle restores original state |
| [`Fill()`](../internal/canvas/canvas.go:117) / [`Reset()`](../internal/canvas/canvas.go:126) | Idempotence | Multiple calls have same effect as one |
| [`charAt()`](../internal/canvas/canvas.go:135) | Braille encoding | Character always in valid braille range |

### 2. History Package - [`internal/history/message.go`](../internal/history/message.go:1)

Message conversion functions have clear round-trip properties.

#### Properties to Test

| Function | Property | Description |
|----------|----------|-------------|
| [`ToOpenAI()`](../internal/history/message.go:56) / [`MessageFromOpenAI()`](../internal/history/message.go:64) | Round-trip | Converting to OpenAI and back preserves Role and Content |
| [`MessagesToOpenAI()`](../internal/history/message.go:72) | Length preservation | Output slice length equals input slice length |
| [`IsSummary()`](../internal/history/message.go:46) / [`IsMessage()`](../internal/history/message.go:51) | Mutual exclusion | A message is either a summary or a message, never both |

### 3. Config Package - [`internal/config/config.go`](../internal/config/config.go:1)

Configuration defaults and validation have invariant properties.

#### Properties to Test

| Function | Property | Description |
|----------|----------|-------------|
| [`applyDefaults()`](../internal/config/config.go:113) | Non-empty fields | After defaults, Model, MaxTokens, Temperature, BaseURL are non-zero |
| [`applyDefaults()`](../internal/config/config.go:113) | Idempotence | Applying defaults twice equals applying once |
| [`validate()`](../internal/config/config.go:160) | Determinism | Same config always produces same validation result |

## Example Property Tests

### Canvas: Dimension Rounding Property

```go
package canvas

import (
    "testing"
    "testing/quick"
)

// TestNewDimensionRounding verifies that New() always rounds dimensions
// to valid braille cell boundaries.
func TestNewDimensionRounding(t *testing.T) {
    property := func(width, height uint8) bool {
        // Use uint8 to limit input range (0-255)
        w, h := int(width), int(height)
        if w == 0 || h == 0 {
            return true // Skip zero dimensions
        }

        c := New(w, h)

        // Property 1: Width is always even
        if c.Width()%2 != 0 {
            return false
        }

        // Property 2: Height is always multiple of 4
        if c.Height()%4 != 0 {
            return false
        }

        // Property 3: Dimensions are >= input
        if c.Width() < w || c.Height() < h {
            return false
        }

        // Property 4: Dimensions don't grow more than necessary
        if c.Width() > w+1 || c.Height() > h+3 {
            return false
        }

        return true
    }

    if err := quick.Check(property, nil); err != nil {
        t.Error(err)
    }
}
```

### Canvas: Set/Get Round-Trip Property

```go
// TestSetGetRoundTrip verifies that Set followed by Get returns true
// for any valid coordinate.
func TestSetGetRoundTrip(t *testing.T) {
    property := func(width, height, x, y uint8) bool {
        w, h := int(width)+2, int(height)+4 // Ensure minimum size
        px, py := int(x), int(y)

        c := New(w, h)

        // Only test in-bounds coordinates
        if px >= c.Width() || py >= c.Height() {
            return true // Skip out-of-bounds
        }

        // Property: Set then Get returns true
        c.Set(px, py)
        return c.Get(px, py)
    }

    if err := quick.Check(property, nil); err != nil {
        t.Error(err)
    }
}
```

### Canvas: Toggle Involution Property

```go
// TestToggleInvolution verifies that toggling twice restores original state.
func TestToggleInvolution(t *testing.T) {
    property := func(width, height, x, y uint8, initialState bool) bool {
        w, h := int(width)+2, int(height)+4
        px, py := int(x), int(y)

        c := New(w, h)

        if px >= c.Width() || py >= c.Height() {
            return true
        }

        // Set initial state
        if initialState {
            c.Set(px, py)
        }
        original := c.Get(px, py)

        // Toggle twice
        c.Toggle(px, py)
        c.Toggle(px, py)

        // Property: State is restored
        return c.Get(px, py) == original
    }

    if err := quick.Check(property, nil); err != nil {
        t.Error(err)
    }
}
```

### Canvas: Braille Character Range Property

```go
// TestBrailleCharacterRange verifies that rendered characters are valid braille.
func TestBrailleCharacterRange(t *testing.T) {
    const brailleBase = '\u2800'
    const brailleMax = '\u28FF'

    property := func(pixels [8]bool) bool {
        c := New(2, 4) // Single braille cell

        // Set pixels based on input
        positions := [][2]int{
            {0, 0}, {0, 1}, {0, 2}, {0, 3},
            {1, 0}, {1, 1}, {1, 2}, {1, 3},
        }
        for i, on := range pixels {
            if on {
                c.Set(positions[i][0], positions[i][1])
            }
        }

        // Get the rendered character
        str := c.String()
        if len(str) == 0 {
            return false
        }

        char := []rune(str)[0]

        // Property: Character is in valid braille range
        return char >= brailleBase && char <= brailleMax
    }

    if err := quick.Check(property, nil); err != nil {
        t.Error(err)
    }
}
```

### History: Message Conversion Round-Trip

```go
package history

import (
    "testing"
    "testing/quick"
)

// TestMessageRoundTrip verifies that converting to OpenAI format and back
// preserves the essential fields.
func TestMessageRoundTrip(t *testing.T) {
    property := func(content string, roleIdx uint8) bool {
        roles := []Role{RoleUser, RoleAssistant, RoleSystem}
        role := roles[int(roleIdx)%len(roles)]

        original := Message{
            Role:    role,
            Content: content,
        }

        // Convert to OpenAI and back
        openaiMsg := original.ToOpenAI()
        restored := MessageFromOpenAI(openaiMsg)

        // Property: Role and Content are preserved
        return restored.Role == original.Role &&
            restored.Content == original.Content
    }

    if err := quick.Check(property, nil); err != nil {
        t.Error(err)
    }
}
```

### History: Message Type Mutual Exclusion

```go
// TestMessageTypeMutualExclusion verifies that IsSummary and IsMessage
// are mutually exclusive for any message type.
func TestMessageTypeMutualExclusion(t *testing.T) {
    property := func(typeIdx uint8) bool {
        types := []MessageType{"", TypeMessage, TypeSummary, "unknown"}
        msgType := types[int(typeIdx)%len(types)]

        m := Message{Type: msgType}

        isSummary := m.IsSummary()
        isMessage := m.IsMessage()

        // Property: Exactly one of IsSummary or IsMessage is true
        // (except for unknown types which should be treated as messages)
        if msgType == TypeSummary {
            return isSummary && !isMessage
        }
        return !isSummary && isMessage
    }

    if err := quick.Check(property, nil); err != nil {
        t.Error(err)
    }
}
```

### Config: Defaults Idempotence

```go
package config

import (
    "testing"
    "testing/quick"
)

// TestApplyDefaultsIdempotence verifies that applying defaults twice
// produces the same result as applying once.
func TestApplyDefaultsIdempotence(t *testing.T) {
    property := func(model string, maxTokens uint16, temp float64) bool {
        // Create two identical configs
        c1 := &Config{
            OpenAI: OpenAIConfig{
                Model:       model,
                MaxTokens:   int(maxTokens),
                Temperature: temp,
            },
        }
        c2 := &Config{
            OpenAI: OpenAIConfig{
                Model:       model,
                MaxTokens:   int(maxTokens),
                Temperature: temp,
            },
        }

        // Apply defaults once to c1
        c1.applyDefaults()

        // Apply defaults twice to c2
        c2.applyDefaults()
        c2.applyDefaults()

        // Property: Results are identical
        return c1.OpenAI.Model == c2.OpenAI.Model &&
            c1.OpenAI.MaxTokens == c2.OpenAI.MaxTokens &&
            c1.OpenAI.Temperature == c2.OpenAI.Temperature &&
            c1.OpenAI.BaseURL == c2.OpenAI.BaseURL
    }

    if err := quick.Check(property, nil); err != nil {
        t.Error(err)
    }
}
```

## Best Practices

### 1. Property Naming Convention

Use descriptive names that state the property being tested:

```go
// Good
func TestToggleInvolution(t *testing.T)
func TestDimensionRoundingToValidBrailleBoundaries(t *testing.T)

// Avoid
func TestToggle(t *testing.T)
func TestNew(t *testing.T)
```

### 2. Handle Edge Cases Explicitly

Skip invalid inputs rather than letting them cause false failures:

```go
property := func(x, y uint8) bool {
    if x == 0 || y == 0 {
        return true // Skip edge case
    }
    // ... actual property test
}
```

### 3. Use Bounded Types

Prefer `uint8` or `uint16` over `int` to limit the input space:

```go
// Good - bounded input
func(width, height uint8) bool

// Avoid - unbounded input can cause issues
func(width, height int) bool
```

### 4. Configure Test Iterations

Use `quick.Config` for more control:

```go
config := &quick.Config{
    MaxCount: 1000,  // Number of iterations
    Rand:     rand.New(rand.NewSource(42)), // Reproducible
}
if err := quick.Check(property, config); err != nil {
    t.Error(err)
}
```

### 5. Document Properties

Add comments explaining what mathematical property is being verified:

```go
// TestToggleInvolution verifies the involution property:
// For any function f, f(f(x)) = x
// Toggle is an involution because toggling twice restores the original state.
func TestToggleInvolution(t *testing.T) {
```

### 6. Combine with Table-Driven Tests

Property tests complement, not replace, example-based tests:

```go
// Example-based test for specific known values
func TestSinglePixelBraille(t *testing.T) {
    tests := []struct{...}{...}
    // ...
}

// Property-based test for general invariants
func TestBrailleCharacterRange(t *testing.T) {
    // ...
}
```

## Helper Utilities

### Custom Generator for Message

For types that `testing/quick` cannot generate automatically, implement the `quick.Generator` interface:

```go
package history

import (
    "math/rand"
    "reflect"
    "testing/quick"
)

// Generate implements quick.Generator for Message.
func (Message) Generate(rand *rand.Rand, size int) reflect.Value {
    roles := []Role{RoleUser, RoleAssistant, RoleSystem}
    types := []MessageType{"", TypeMessage, TypeSummary}

    content, _ := quick.Value(reflect.TypeOf(""), rand)

    m := Message{
        Role:    roles[rand.Intn(len(roles))],
        Content: content.String(),
        Type:    types[rand.Intn(len(types))],
    }

    return reflect.ValueOf(m)
}
```

### Test Helper for Canvas Properties

```go
package canvas

import "testing/quick"

// canvasProperty wraps a property function with common setup.
func canvasProperty(fn func(c *Canvas, x, y int) bool) func(uint8, uint8, uint8, uint8) bool {
    return func(width, height, x, y uint8) bool {
        w, h := int(width)+2, int(height)+4
        px, py := int(x), int(y)

        c := New(w, h)

        if px >= c.Width() || py >= c.Height() {
            return true // Skip out-of-bounds
        }

        return fn(c, px, py)
    }
}

// Usage:
func TestSetGetRoundTrip(t *testing.T) {
    property := canvasProperty(func(c *Canvas, x, y int) bool {
        c.Set(x, y)
        return c.Get(x, y)
    })

    if err := quick.Check(property, nil); err != nil {
        t.Error(err)
    }
}
```

## File Organization

Place property tests alongside existing tests:

```
internal/
├── canvas/
│   ├── canvas.go
│   ├── canvas_test.go          # Existing example-based tests
│   └── canvas_property_test.go # New property-based tests
├── history/
│   ├── message.go
│   └── message_property_test.go
└── config/
    ├── config.go
    └── config_property_test.go
```

## Running Property Tests

Property tests run with standard `go test`:

```bash
# Run all tests including property tests
go test ./...

# Run with verbose output to see property test iterations
go test -v ./internal/canvas/...

# Run specific property test
go test -v -run TestToggleInvolution ./internal/canvas/...
```

## Summary

| Aspect | Recommendation |
|--------|----------------|
| Library | `testing/quick` (stdlib) |
| Primary target | `internal/canvas` package |
| Secondary targets | `internal/history`, `internal/config` |
| Test file naming | `*_property_test.go` |
| Integration | Standard `go test ./...` |

This approach provides robust property-based testing with zero additional dependencies, following Go idioms and integrating seamlessly with the existing test infrastructure.
