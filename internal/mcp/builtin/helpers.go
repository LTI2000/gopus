package builtin

import (
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// GetArgs extracts the arguments map from a CallToolRequest.
// Returns an error if the arguments are not in the expected format.
func GetArgs(req mcplib.CallToolRequest) (map[string]any, error) {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}
	return args, nil
}

// GetStringArg extracts a required string argument from the arguments map.
// Returns an error if the argument is missing or not a string.
func GetStringArg(args map[string]any, name string) (string, error) {
	val, ok := args[name].(string)
	if !ok {
		return "", fmt.Errorf("%s argument is required and must be a string", name)
	}
	return val, nil
}

// GetOptionalStringArg extracts an optional string argument from the arguments map.
// Returns the default value if the argument is missing or not a string.
func GetOptionalStringArg(args map[string]any, name string, defaultVal string) string {
	if val, ok := args[name].(string); ok && val != "" {
		return val
	}
	return defaultVal
}

// GetRequiredStringArg is a convenience function that combines GetArgs and GetStringArg.
// It extracts a required string argument directly from a CallToolRequest.
func GetRequiredStringArg(req mcplib.CallToolRequest, name string) (string, error) {
	args, err := GetArgs(req)
	if err != nil {
		return "", err
	}
	return GetStringArg(args, name)
}
