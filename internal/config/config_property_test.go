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

// TestApplyDefaultsNonEmptyFields verifies that after applying defaults,
// critical fields are non-zero.
func TestApplyDefaultsNonEmptyFields(t *testing.T) {
	property := func(model string, maxTokens uint16, temp float64, baseURL string) bool {
		c := &Config{
			OpenAI: OpenAIConfig{
				Model:       model,
				MaxTokens:   int(maxTokens),
				Temperature: temp,
				BaseURL:     baseURL,
			},
		}

		c.applyDefaults()

		// Property: After defaults, these fields are non-zero
		if c.OpenAI.Model == "" {
			return false
		}
		if c.OpenAI.MaxTokens == 0 {
			return false
		}
		if c.OpenAI.Temperature == 0 {
			return false
		}
		if c.OpenAI.BaseURL == "" {
			return false
		}

		return true
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestApplyDefaultsPreservesExistingValues verifies that applyDefaults
// does not overwrite non-zero values.
func TestApplyDefaultsPreservesExistingValues(t *testing.T) {
	property := func(model string, maxTokens uint16, temp float64, baseURL string) bool {
		// Skip if all values are zero/empty (defaults would apply)
		if model == "" && maxTokens == 0 && temp == 0 && baseURL == "" {
			return true
		}

		c := &Config{
			OpenAI: OpenAIConfig{
				Model:       model,
				MaxTokens:   int(maxTokens),
				Temperature: temp,
				BaseURL:     baseURL,
			},
		}

		c.applyDefaults()

		// Property: Non-zero values are preserved
		if model != "" && c.OpenAI.Model != model {
			return false
		}
		if maxTokens != 0 && c.OpenAI.MaxTokens != int(maxTokens) {
			return false
		}
		if temp != 0 && c.OpenAI.Temperature != temp {
			return false
		}
		if baseURL != "" && c.OpenAI.BaseURL != baseURL {
			return false
		}

		return true
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestValidateDeterminism verifies that validate produces the same result
// for the same config.
func TestValidateDeterminism(t *testing.T) {
	property := func(apiKey string) bool {
		c1 := &Config{
			OpenAI: OpenAIConfig{
				APIKey: apiKey,
			},
		}
		c2 := &Config{
			OpenAI: OpenAIConfig{
				APIKey: apiKey,
			},
		}

		err1 := c1.validate()
		err2 := c2.validate()

		// Property: Same config produces same validation result
		if err1 == nil && err2 == nil {
			return true
		}
		if err1 != nil && err2 != nil {
			return err1.Error() == err2.Error()
		}
		return false
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestValidateRequiresAPIKey verifies that validate fails when APIKey is empty.
func TestValidateRequiresAPIKey(t *testing.T) {
	property := func(model string, maxTokens uint16) bool {
		c := &Config{
			OpenAI: OpenAIConfig{
				APIKey:    "", // Empty API key
				Model:     model,
				MaxTokens: int(maxTokens),
			},
		}

		err := c.validate()

		// Property: Empty API key causes validation error
		return err != nil
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestValidatePassesWithAPIKey verifies that validate passes when APIKey is set.
func TestValidatePassesWithAPIKey(t *testing.T) {
	property := func(apiKey string) bool {
		if apiKey == "" {
			return true // Skip empty API keys
		}

		c := &Config{
			OpenAI: OpenAIConfig{
				APIKey: apiKey,
			},
		}

		err := c.validate()

		// Property: Non-empty API key passes validation
		return err == nil
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestSummarizationDefaultsIdempotence verifies that applying summarization
// defaults twice produces the same result as applying once.
func TestSummarizationDefaultsIdempotence(t *testing.T) {
	property := func(recentCount, condensedCount uint8) bool {
		c1 := &Config{
			Summarization: SummarizationConfig{
				RecentCount:    int(recentCount),
				CondensedCount: int(condensedCount),
			},
		}
		c2 := &Config{
			Summarization: SummarizationConfig{
				RecentCount:    int(recentCount),
				CondensedCount: int(condensedCount),
			},
		}

		// Apply defaults once to c1
		c1.applySummarizationDefaults()

		// Apply defaults twice to c2
		c2.applySummarizationDefaults()
		c2.applySummarizationDefaults()

		// Property: Results are identical
		return c1.Summarization.Enabled == c2.Summarization.Enabled &&
			c1.Summarization.RecentCount == c2.Summarization.RecentCount &&
			c1.Summarization.CondensedCount == c2.Summarization.CondensedCount &&
			c1.Summarization.AutoSummarize == c2.Summarization.AutoSummarize &&
			c1.Summarization.AutoThreshold == c2.Summarization.AutoThreshold &&
			c1.Summarization.CondensedPrompt == c2.Summarization.CondensedPrompt &&
			c1.Summarization.CompressedPrompt == c2.Summarization.CompressedPrompt
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}

// TestSummarizationDefaultsNonEmptyPrompts verifies that after applying
// defaults, prompts are non-empty.
func TestSummarizationDefaultsNonEmptyPrompts(t *testing.T) {
	property := func(condensedPrompt, compressedPrompt string) bool {
		c := &Config{
			Summarization: SummarizationConfig{
				CondensedPrompt:  condensedPrompt,
				CompressedPrompt: compressedPrompt,
			},
		}

		c.applySummarizationDefaults()

		// Property: After defaults, prompts are non-empty
		return c.Summarization.CondensedPrompt != "" &&
			c.Summarization.CompressedPrompt != ""
	}

	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}
