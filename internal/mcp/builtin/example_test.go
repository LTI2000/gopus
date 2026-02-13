package builtin

import (
	"context"
	"strconv"
	"testing"
	"testing/quick"
	"time"

	"gopus/internal/mcp"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// getToolHandler retrieves a registered tool handler by name from the DefaultToolRegistry.
func getToolHandler(name string) mcp.ToolHandler {
	reg, ok := mcp.DefaultToolRegistry.Get(name)
	if !ok {
		return nil
	}
	return reg.HandlerFactory(nil) // Pass nil for openaiClient since these tools don't use it
}

// getToolRegistration retrieves a tool registration by name.
func getToolRegistration(name string) (mcp.ToolRegistration, bool) {
	return mcp.DefaultToolRegistry.Get(name)
}

// TestEchoTool tests the echo tool functionality.
func TestEchoTool(t *testing.T) {
	handler := getToolHandler("echo")
	if handler == nil {
		t.Fatal("echo tool not found in registry")
	}

	tests := []struct {
		name    string
		args    map[string]any
		want    string
		wantErr bool
	}{
		{
			name:    "simple message",
			args:    map[string]any{"message": "hello"},
			want:    "Echo: hello",
			wantErr: false,
		},
		{
			name:    "empty message",
			args:    map[string]any{"message": ""},
			want:    "Echo: ",
			wantErr: false,
		},
		{
			name:    "unicode message",
			args:    map[string]any{"message": "こんにちは 🎉"},
			want:    "Echo: こんにちは 🎉",
			wantErr: false,
		},
		{
			name:    "message with newlines",
			args:    map[string]any{"message": "line1\nline2\nline3"},
			want:    "Echo: line1\nline2\nline3",
			wantErr: false,
		},
		{
			name:    "message with special characters",
			args:    map[string]any{"message": "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
			want:    "Echo: !@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr: false,
		},
		{
			name:    "missing message argument",
			args:    map[string]any{},
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type for message",
			args:    map[string]any{"message": 42},
			want:    "",
			wantErr: true,
		},
		{
			name:    "nil message",
			args:    map[string]any{"message": nil},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeCallToolRequest(tt.args)
			result, err := handler(context.Background(), req)

			if (err != nil) != tt.wantErr {
				t.Errorf("echo handler error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				text, ok := getTextContent(result)
				if !ok {
					t.Fatal("expected TextContent result")
				}
				if text != tt.want {
					t.Errorf("echo result = %q, want %q", text, tt.want)
				}
			}
		})
	}
}

// TestCurrentTimeTool tests the current_time tool functionality.
func TestCurrentTimeTool(t *testing.T) {
	handler := getToolHandler("current_time")
	if handler == nil {
		t.Fatal("current_time tool not found in registry")
	}

	tests := []struct {
		name     string
		args     map[string]any
		validate func(t *testing.T, result string)
		wantErr  bool
	}{
		{
			name: "default format (RFC3339)",
			args: map[string]any{},
			validate: func(t *testing.T, result string) {
				_, err := time.Parse(time.RFC3339, result)
				if err != nil {
					t.Errorf("expected RFC3339 format, got %q: %v", result, err)
				}
			},
			wantErr: false,
		},
		{
			name: "explicit RFC3339 format",
			args: map[string]any{"format": "RFC3339"},
			validate: func(t *testing.T, result string) {
				_, err := time.Parse(time.RFC3339, result)
				if err != nil {
					t.Errorf("expected RFC3339 format, got %q: %v", result, err)
				}
			},
			wantErr: false,
		},
		{
			name: "iso format",
			args: map[string]any{"format": "iso"},
			validate: func(t *testing.T, result string) {
				_, err := time.Parse(time.RFC3339, result)
				if err != nil {
					t.Errorf("expected ISO8601/RFC3339 format, got %q: %v", result, err)
				}
			},
			wantErr: false,
		},
		{
			name: "ISO8601 format",
			args: map[string]any{"format": "ISO8601"},
			validate: func(t *testing.T, result string) {
				_, err := time.Parse(time.RFC3339, result)
				if err != nil {
					t.Errorf("expected ISO8601/RFC3339 format, got %q: %v", result, err)
				}
			},
			wantErr: false,
		},
		{
			name: "unix timestamp format",
			args: map[string]any{"format": "unix"},
			validate: func(t *testing.T, result string) {
				ts, err := strconv.ParseInt(result, 10, 64)
				if err != nil {
					t.Errorf("expected unix timestamp, got %q: %v", result, err)
					return
				}
				// Verify timestamp is reasonable (between 2020 and 2100)
				if ts < 1577836800 || ts > 4102444800 {
					t.Errorf("unix timestamp %d seems unreasonable", ts)
				}
			},
			wantErr: false,
		},
		{
			name: "custom Go format",
			args: map[string]any{"format": "2006-01-02"},
			validate: func(t *testing.T, result string) {
				_, err := time.Parse("2006-01-02", result)
				if err != nil {
					t.Errorf("expected date format YYYY-MM-DD, got %q: %v", result, err)
				}
			},
			wantErr: false,
		},
		{
			name: "custom time format",
			args: map[string]any{"format": "15:04:05"},
			validate: func(t *testing.T, result string) {
				_, err := time.Parse("15:04:05", result)
				if err != nil {
					t.Errorf("expected time format HH:MM:SS, got %q: %v", result, err)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeCallToolRequest(tt.args)
			result, err := handler(context.Background(), req)

			if (err != nil) != tt.wantErr {
				t.Errorf("current_time handler error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				text, ok := getTextContent(result)
				if !ok {
					t.Fatal("expected TextContent result")
				}
				tt.validate(t, text)
			}
		})
	}
}

// TestCurrentTimeToolTimeBounds verifies that current_time returns a time
// within a reasonable window of the actual current time.
func TestCurrentTimeToolTimeBounds(t *testing.T) {
	handler := getToolHandler("current_time")
	if handler == nil {
		t.Fatal("current_time tool not found in registry")
	}

	before := time.Now().Add(-time.Second)
	req := makeCallToolRequest(map[string]any{"format": "unix"})
	result, err := handler(context.Background(), req)
	after := time.Now().Add(time.Second)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text, ok := getTextContent(result)
	if !ok {
		t.Fatal("expected TextContent result")
	}

	ts, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		t.Fatalf("failed to parse unix timestamp: %v", err)
	}

	resultTime := time.Unix(ts, 0)
	if resultTime.Before(before) || resultTime.After(after) {
		t.Errorf("result time %v not within bounds [%v, %v]", resultTime, before, after)
	}
}

// Property-based tests

// TestEchoToolProperty verifies that the echo tool always prefixes
// the message with "Echo: " for any valid string input.
func TestEchoToolProperty(t *testing.T) {
	handler := getToolHandler("echo")
	if handler == nil {
		t.Fatal("echo tool not found in registry")
	}

	property := func(message string) bool {
		req := makeCallToolRequest(map[string]any{"message": message})
		result, err := handler(context.Background(), req)
		if err != nil {
			return false
		}
		text, ok := getTextContent(result)
		if !ok {
			return false
		}
		expected := "Echo: " + message
		return text == expected
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestCurrentTimeToolUnixProperty verifies that unix format always returns
// a valid integer timestamp.
func TestCurrentTimeToolUnixProperty(t *testing.T) {
	handler := getToolHandler("current_time")
	if handler == nil {
		t.Fatal("current_time tool not found in registry")
	}

	// Run multiple times to check consistency
	for i := 0; i < 100; i++ {
		req := makeCallToolRequest(map[string]any{"format": "unix"})
		result, err := handler(context.Background(), req)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		text, ok := getTextContent(result)
		if !ok {
			t.Fatalf("iteration %d: expected TextContent", i)
		}
		_, err = strconv.ParseInt(text, 10, 64)
		if err != nil {
			t.Fatalf("iteration %d: failed to parse as int: %v", i, err)
		}
	}
}

// TestCurrentTimeToolRFC3339Property verifies that RFC3339 format always
// returns a valid RFC3339 timestamp.
func TestCurrentTimeToolRFC3339Property(t *testing.T) {
	handler := getToolHandler("current_time")
	if handler == nil {
		t.Fatal("current_time tool not found in registry")
	}

	formats := []string{"RFC3339", "iso", "ISO8601", ""}
	for _, format := range formats {
		args := map[string]any{}
		if format != "" {
			args["format"] = format
		}
		req := makeCallToolRequest(args)
		result, err := handler(context.Background(), req)
		if err != nil {
			t.Fatalf("format %q: unexpected error: %v", format, err)
		}
		textContent, ok := result.Content[0].(mcplib.TextContent)
		if !ok {
			t.Fatalf("format %q: expected TextContent", format)
		}
		_, err = time.Parse(time.RFC3339, textContent.Text)
		if err != nil {
			t.Fatalf("format %q: failed to parse as RFC3339: %v", format, err)
		}
	}
}

// TestCurrentTimeToolMonotonicProperty verifies that successive calls
// return non-decreasing timestamps.
func TestCurrentTimeToolMonotonicProperty(t *testing.T) {
	handler := getToolHandler("current_time")
	if handler == nil {
		t.Fatal("current_time tool not found in registry")
	}

	var prevTs int64 = 0
	for i := 0; i < 100; i++ {
		req := makeCallToolRequest(map[string]any{"format": "unix"})
		result, err := handler(context.Background(), req)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		textContent, ok := result.Content[0].(mcplib.TextContent)
		if !ok {
			t.Fatalf("iteration %d: expected TextContent", i)
		}
		ts, err := strconv.ParseInt(textContent.Text, 10, 64)
		if err != nil {
			t.Fatalf("iteration %d: failed to parse as int: %v", i, err)
		}
		if ts < prevTs {
			t.Fatalf("iteration %d: timestamp %d < previous %d (not monotonic)", i, ts, prevTs)
		}
		prevTs = ts
	}
}

// TestToolRegistration verifies that both tools are properly registered.
func TestToolRegistration(t *testing.T) {
	expectedTools := []string{"echo", "current_time"}

	for _, name := range expectedTools {
		_, ok := mcp.DefaultToolRegistry.Get(name)
		if !ok {
			t.Errorf("expected tool %q to be registered", name)
		}
	}
}

// TestEchoToolDefinition verifies the echo tool has correct metadata.
func TestEchoToolDefinition(t *testing.T) {
	reg, ok := getToolRegistration("echo")
	if !ok {
		t.Fatal("echo tool not found")
	}

	echoTool := reg.Tool

	if echoTool.Description != "Echoes back the input message" {
		t.Errorf("unexpected description: %q", echoTool.Description)
	}

	// Verify the tool has the message parameter defined
	schema := echoTool.InputSchema
	props := schema.Properties

	messageProp, ok := props["message"]
	if !ok {
		t.Fatal("expected 'message' property")
	}

	messageSchema, ok := messageProp.(map[string]any)
	if !ok {
		t.Fatal("expected message property to be a map")
	}

	if messageSchema["type"] != "string" {
		t.Errorf("expected message type to be 'string', got %v", messageSchema["type"])
	}

	// Check that message is required
	required := schema.Required

	found := false
	for _, r := range required {
		if r == "message" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'message' to be in required list")
	}
}

// TestCurrentTimeToolDefinition verifies the current_time tool has correct metadata.
func TestCurrentTimeToolDefinition(t *testing.T) {
	reg, ok := getToolRegistration("current_time")
	if !ok {
		t.Fatal("current_time tool not found")
	}

	timeTool := reg.Tool

	if timeTool.Description != "Returns the current date and time" {
		t.Errorf("unexpected description: %q", timeTool.Description)
	}

	// Verify the tool has the format parameter defined
	schema := timeTool.InputSchema
	props := schema.Properties

	formatProp, ok := props["format"]
	if !ok {
		t.Fatal("expected 'format' property")
	}

	formatSchema, ok := formatProp.(map[string]any)
	if !ok {
		t.Fatal("expected format property to be a map")
	}

	if formatSchema["type"] != "string" {
		t.Errorf("expected format type to be 'string', got %v", formatSchema["type"])
	}

	// format should NOT be required (it's optional)
	required := schema.Required
	for _, r := range required {
		if r == "format" {
			t.Error("'format' should not be required")
		}
	}
}
