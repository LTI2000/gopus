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
	OpenAI  OpenAIConfig  `yaml:"openai"`
	History HistoryConfig `yaml:"history"`
}

// HistoryConfig contains chat history settings.
type HistoryConfig struct {
	SessionsDir string `yaml:"sessions_dir"`
}

// OpenAIConfig contains OpenAI API settings.
type OpenAIConfig struct {
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
	BaseURL     string  `yaml:"base_url"`
}

// DefaultConfigPath is the default path to look for the configuration file.
const DefaultConfigPath = "config.yaml"

// Default values for optional configuration fields.
const (
	DefaultModel       = "gpt-3.5-turbo"
	DefaultMaxTokens   = 1000
	DefaultTemperature = 0.7
	DefaultBaseURL     = "https://api.openai.com/v1"
)

// Load reads and parses the configuration from the specified file path.
func Load(path string) (*Config, error) {
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

	return &cfg, nil
}

// LoadDefault loads configuration from the default path (config.yaml).
func LoadDefault() (*Config, error) {
	return Load(DefaultConfigPath)
}

// applyDefaults sets default values for optional configuration fields.
func (c *Config) applyDefaults() {
	if c.OpenAI.Model == "" {
		c.OpenAI.Model = DefaultModel
	}
	if c.OpenAI.MaxTokens == 0 {
		c.OpenAI.MaxTokens = DefaultMaxTokens
	}
	if c.OpenAI.Temperature == 0 {
		c.OpenAI.Temperature = DefaultTemperature
	}
	if c.OpenAI.BaseURL == "" {
		c.OpenAI.BaseURL = DefaultBaseURL
	}
}

// validate checks that all required configuration fields are present.
func (c *Config) validate() error {
	if c.OpenAI.APIKey == "" {
		return errors.New("openai.api_key is required in configuration")
	}
	return nil
}
