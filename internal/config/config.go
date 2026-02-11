// Package config provides configuration loading and validation for the chat application.
package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	OpenAI        OpenAIConfig        `yaml:"openai"`
	History       HistoryConfig       `yaml:"history"`
	Summarization SummarizationConfig `yaml:"summarization"`
	MCP           MCPConfig           `yaml:"mcp"`
}

// HistoryConfig contains chat history settings.
type HistoryConfig struct {
	SessionsDir string `yaml:"sessions_dir"`
}

// SummarizationConfig contains settings for automatic history summarization.
type SummarizationConfig struct {
	Enabled          bool   `yaml:"enabled"`           // Enable summarization feature
	RecentCount      int    `yaml:"recent_count"`      // Messages to keep in full detail
	CondensedCount   int    `yaml:"condensed_count"`   // Messages to condense before compressing
	AutoSummarize    bool   `yaml:"auto_summarize"`    // Enable automatic summarization
	AutoThreshold    int    `yaml:"auto_threshold"`    // Trigger auto-summarization when message count exceeds this
	CondensedPrompt  string `yaml:"condensed_prompt"`  // Prompt for condensed summarization
	CompressedPrompt string `yaml:"compressed_prompt"` // Prompt for compressed summarization
}

// OpenAIConfig contains OpenAI API settings.
type OpenAIConfig struct {
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
	BaseURL     string  `yaml:"base_url"`
}

// MCPConfig contains MCP client settings.
type MCPConfig struct {
	Enabled          bool              `yaml:"enabled"`           // Enable MCP support
	ToolConfirmation string            `yaml:"tool_confirmation"` // "always", "never", or "ask"
	DefaultTimeout   int               `yaml:"default_timeout"`   // Timeout in seconds for MCP requests
	Servers          []MCPServerConfig `yaml:"servers"`           // List of MCP servers to connect to
}

// MCPServerConfig defines an MCP server connection.
type MCPServerConfig struct {
	Name    string            `yaml:"name"`     // Unique identifier for this server
	Command string            `yaml:"command"`  // Command to start the server
	Args    []string          `yaml:"args"`     // Command arguments
	Env     map[string]string `yaml:"env"`      // Additional environment variables
	WorkDir string            `yaml:"work_dir"` // Working directory for the command
	Enabled bool              `yaml:"enabled"`  // Enable/disable this server
}

// ToolConfirmation constants
const (
	ToolConfirmationAlways = "always" // Always ask before executing tools
	ToolConfirmationNever  = "never"  // Never ask, execute automatically
	ToolConfirmationAsk    = "ask"    // Ask based on tool risk level (default)
)

// DefaultConfigPath is the default path to look for the configuration file.
const DefaultConfigPath = "config.yaml"

// default values for optional configuration fields.
const (
	defaultModel       = "gpt-3.5-turbo"
	defaultMaxTokens   = 1000
	defaultTemperature = 0.7
	defaultBaseURL     = "https://api.openai.com/v1"

	// Summarization defaults
	defaultSummarizationEnabled        = true
	defaultSummarizationRecentCount    = 20
	defaultSummarizationCondensedCount = 50
	defaultSummarizationAutoSummarize  = true
	defaultSummarizationAutoThreshold  = 100

	// MCP defaults
	defaultMCPEnabled          = false // Disabled by default until servers are configured
	defaultMCPToolConfirmation = ToolConfirmationAsk
	defaultMCPDefaultTimeout   = 30 // seconds
)

// Default prompts for summarization.
const (
	DefaultCondensedPrompt = `Summarize the following conversation, preserving:
- Key topics discussed
- Important decisions or conclusions
- Relevant context for future conversations

Keep the summary concise but informative. Write in third person, describing what the user and assistant discussed.`

	DefaultCompressedPrompt = `Create a highly compressed summary of this conversation history.
Include only:
- Main topics covered
- Critical facts or decisions
- Essential context

Be extremely brief - this is long-term memory. Write in third person.`
)

// Load reads and parses the configuration from the specified file path.
func Load(path string) (*Config, error) {
	fmt.Printf("Loading configuration from %s...\n", path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for optional fields
	cfg.applyDefaults()

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	fmt.Printf("Using AI model: %s\n", cfg.OpenAI.Model)

	return &cfg, nil
}

// LoadDefault loads configuration from the default path (config.yaml).
func LoadDefault() (*Config, error) {
	return Load(DefaultConfigPath)
}

// applyDefaults sets default values for optional configuration fields.
func (c *Config) applyDefaults() {
	if c.OpenAI.Model == "" {
		c.OpenAI.Model = defaultModel
	}
	if c.OpenAI.MaxTokens == 0 {
		c.OpenAI.MaxTokens = defaultMaxTokens
	}
	if c.OpenAI.Temperature == 0 {
		c.OpenAI.Temperature = defaultTemperature
	}
	if c.OpenAI.BaseURL == "" {
		c.OpenAI.BaseURL = defaultBaseURL
	}

	// Summarization defaults - use a flag to detect if section was present
	c.applySummarizationDefaults()

	// MCP defaults
	c.applyMCPDefaults()
}

// applySummarizationDefaults sets default values for summarization config.
func (c *Config) applySummarizationDefaults() {
	// If RecentCount is 0, apply all defaults (section was likely not specified)
	if c.Summarization.RecentCount == 0 {
		c.Summarization.Enabled = defaultSummarizationEnabled
		c.Summarization.RecentCount = defaultSummarizationRecentCount
		c.Summarization.CondensedCount = defaultSummarizationCondensedCount
		c.Summarization.AutoSummarize = defaultSummarizationAutoSummarize
		c.Summarization.AutoThreshold = defaultSummarizationAutoThreshold
	} else {
		// Section was specified, only fill in missing values
		if c.Summarization.CondensedCount == 0 {
			c.Summarization.CondensedCount = defaultSummarizationCondensedCount
		}
		if c.Summarization.AutoThreshold == 0 {
			c.Summarization.AutoThreshold = defaultSummarizationAutoThreshold
		}
	}

	// Apply default prompts if not specified
	if c.Summarization.CondensedPrompt == "" {
		c.Summarization.CondensedPrompt = DefaultCondensedPrompt
	}
	if c.Summarization.CompressedPrompt == "" {
		c.Summarization.CompressedPrompt = DefaultCompressedPrompt
	}
}

// applyMCPDefaults sets default values for MCP config.
func (c *Config) applyMCPDefaults() {
	// Apply defaults for tool confirmation
	if c.MCP.ToolConfirmation == "" {
		c.MCP.ToolConfirmation = defaultMCPToolConfirmation
	}

	// Apply default timeout
	if c.MCP.DefaultTimeout == 0 {
		c.MCP.DefaultTimeout = defaultMCPDefaultTimeout
	}

	// Enable MCP if servers are configured
	if len(c.MCP.Servers) > 0 && !c.MCP.Enabled {
		// Don't auto-enable, but this is where we could
	}

	// Note: We don't auto-enable servers because YAML unmarshaling sets bool
	// to false by default, so we can't distinguish between "not set" and
	// "explicitly set to false". Users must set enabled: true explicitly.
}

// validate checks that all required configuration fields are present.
func (c *Config) validate() error {
	if c.OpenAI.APIKey == "" {
		return errors.New("openai.api_key is required in configuration")
	}
	return nil
}
