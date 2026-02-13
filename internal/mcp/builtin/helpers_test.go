package builtin

import (
	"testing"
	"testing/quick"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// makeCallToolRequest creates a CallToolRequest with the given arguments.
func makeCallToolRequest(args any) mcplib.CallToolRequest {
	return mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name:      "test_tool",
			Arguments: args,
		},
	}
}

// getTextContent extracts the text content from a CallToolResult.
// Returns the text and true if successful, or empty string and false if not.
func getTextContent(result *mcplib.CallToolResult) (string, bool) {
	if result == nil || len(result.Content) == 0 {
		return "", false
	}
	textContent, ok := result.Content[0].(mcplib.TextContent)
	if !ok {
		return "", false
	}
	return textContent.Text, true
}

// TestGetArgs tests the GetArgs function with various argument types.
func TestGetArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      any
		wantErr   bool
		wantEmpty bool
	}{
		{
			name:      "valid map with string values",
			args:      map[string]any{"key": "value"},
			wantErr:   false,
			wantEmpty: false,
		},
		{
			name:      "valid empty map",
			args:      map[string]any{},
			wantErr:   false,
			wantEmpty: true,
		},
		{
			name:    "nil arguments",
			args:    nil,
			wantErr: true,
		},
		{
			name:    "string instead of map",
			args:    "invalid",
			wantErr: true,
		},
		{
			name:    "slice instead of map",
			args:    []string{"a", "b"},
			wantErr: true,
		},
		{
			name:    "int instead of map",
			args:    42,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeCallToolRequest(tt.args)

			got, err := GetArgs(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.wantEmpty && len(got) != 0 {
					t.Errorf("GetArgs() = %v, want empty map", got)
				}
				if !tt.wantEmpty && len(got) == 0 {
					t.Errorf("GetArgs() = %v, want non-empty map", got)
				}
			}
		})
	}
}

// TestGetStringArg tests the GetStringArg function.
func TestGetStringArg(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		argName string
		want    string
		wantErr bool
	}{
		{
			name:    "valid string argument",
			args:    map[string]any{"message": "hello"},
			argName: "message",
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "empty string argument",
			args:    map[string]any{"message": ""},
			argName: "message",
			want:    "",
			wantErr: false,
		},
		{
			name:    "missing argument",
			args:    map[string]any{},
			argName: "message",
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type - int",
			args:    map[string]any{"message": 42},
			argName: "message",
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type - bool",
			args:    map[string]any{"message": true},
			argName: "message",
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type - nil",
			args:    map[string]any{"message": nil},
			argName: "message",
			want:    "",
			wantErr: true,
		},
		{
			name:    "unicode string",
			args:    map[string]any{"message": "こんにちは 🎉"},
			argName: "message",
			want:    "こんにちは 🎉",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStringArg(tt.args, tt.argName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStringArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetStringArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetOptionalStringArg tests the GetOptionalStringArg function.
func TestGetOptionalStringArg(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		argName    string
		defaultVal string
		want       string
	}{
		{
			name:       "present string argument",
			args:       map[string]any{"format": "unix"},
			argName:    "format",
			defaultVal: "RFC3339",
			want:       "unix",
		},
		{
			name:       "missing argument returns default",
			args:       map[string]any{},
			argName:    "format",
			defaultVal: "RFC3339",
			want:       "RFC3339",
		},
		{
			name:       "empty string returns default",
			args:       map[string]any{"format": ""},
			argName:    "format",
			defaultVal: "RFC3339",
			want:       "RFC3339",
		},
		{
			name:       "wrong type returns default",
			args:       map[string]any{"format": 123},
			argName:    "format",
			defaultVal: "RFC3339",
			want:       "RFC3339",
		},
		{
			name:       "nil value returns default",
			args:       map[string]any{"format": nil},
			argName:    "format",
			defaultVal: "RFC3339",
			want:       "RFC3339",
		},
		{
			name:       "empty default with missing arg",
			args:       map[string]any{},
			argName:    "format",
			defaultVal: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetOptionalStringArg(tt.args, tt.argName, tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetOptionalStringArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetRequiredStringArg tests the GetRequiredStringArg function.
func TestGetRequiredStringArg(t *testing.T) {
	tests := []struct {
		name    string
		args    any
		argName string
		want    string
		wantErr bool
	}{
		{
			name:    "valid string argument",
			args:    map[string]any{"message": "hello world"},
			argName: "message",
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "missing argument",
			args:    map[string]any{},
			argName: "message",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid arguments format",
			args:    "not a map",
			argName: "message",
			want:    "",
			wantErr: true,
		},
		{
			name:    "nil arguments",
			args:    nil,
			argName: "message",
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type value",
			args:    map[string]any{"message": 42},
			argName: "message",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeCallToolRequest(tt.args)

			got, err := GetRequiredStringArg(req, tt.argName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRequiredStringArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetRequiredStringArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Property-based tests

// TestGetStringArgProperty verifies that GetStringArg always returns the exact
// string value when the argument is present and is a string.
func TestGetStringArgProperty(t *testing.T) {
	property := func(value string) bool {
		args := map[string]any{"key": value}
		got, err := GetStringArg(args, "key")
		if err != nil {
			return false
		}
		return got == value
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestGetOptionalStringArgPropertyPresent verifies that GetOptionalStringArg
// returns the actual value when present and non-empty.
func TestGetOptionalStringArgPropertyPresent(t *testing.T) {
	property := func(value, defaultVal string) bool {
		if value == "" {
			return true // Skip empty values, tested separately
		}
		args := map[string]any{"key": value}
		got := GetOptionalStringArg(args, "key", defaultVal)
		return got == value
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestGetOptionalStringArgPropertyMissing verifies that GetOptionalStringArg
// returns the default value when the argument is missing.
func TestGetOptionalStringArgPropertyMissing(t *testing.T) {
	property := func(defaultVal string) bool {
		args := map[string]any{}
		got := GetOptionalStringArg(args, "missing_key", defaultVal)
		return got == defaultVal
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestGetRequiredStringArgProperty verifies that GetRequiredStringArg
// correctly extracts string values from valid requests.
func TestGetRequiredStringArgProperty(t *testing.T) {
	property := func(value string) bool {
		req := makeCallToolRequest(map[string]any{"key": value})
		got, err := GetRequiredStringArg(req, "key")
		if err != nil {
			return false
		}
		return got == value
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestGetArgsRoundTrip verifies that GetArgs preserves all key-value pairs.
func TestGetArgsRoundTrip(t *testing.T) {
	property := func(key1, val1, key2, val2 string) bool {
		if key1 == key2 {
			return true // Skip duplicate keys
		}
		original := map[string]any{key1: val1, key2: val2}
		req := makeCallToolRequest(original)
		got, err := GetArgs(req)
		if err != nil {
			return false
		}
		return got[key1] == val1 && got[key2] == val2
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}
