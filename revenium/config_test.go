package revenium

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadFromEnv(t *testing.T) {
	// Save original env vars
	originalAPIKey := os.Getenv("REVENIUM_METERING_API_KEY")
	originalOpenAIKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		os.Setenv("REVENIUM_METERING_API_KEY", originalAPIKey)
		os.Setenv("OPENAI_API_KEY", originalOpenAIKey)
	}()

	// Set test env vars
	os.Setenv("REVENIUM_METERING_API_KEY", "hak_test_key_123")
	os.Setenv("OPENAI_API_KEY", "sk-test-openai-key")

	cfg := &Config{}
	err := cfg.loadFromEnv()

	require.NoError(t, err)
	assert.Equal(t, "hak_test_key_123", cfg.ReveniumAPIKey)
	assert.Equal(t, "sk-test-openai-key", cfg.OpenAIAPIKey)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ReveniumAPIKey: "hak_valid_key",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &Config{
				ReveniumAPIKey: "",
			},
			wantErr: true,
		},
		{
			name: "invalid API key format",
			config: &Config{
				ReveniumAPIKey: "invalid_key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNormalizeReveniumBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "https://api.revenium.ai",
		},
		{
			name:     "with trailing slash",
			input:    "https://api.revenium.ai/",
			expected: "https://api.revenium.ai",
		},
		{
			name:     "with /meter/v2",
			input:    "https://api.revenium.ai/meter/v2",
			expected: "https://api.revenium.ai",
		},
		{
			name:     "with /meter",
			input:    "https://api.revenium.ai/meter",
			expected: "https://api.revenium.ai",
		},
		{
			name:     "with /v2",
			input:    "https://api.revenium.ai/v2",
			expected: "https://api.revenium.ai",
		},
		{
			name:     "clean URL",
			input:    "https://api.revenium.ai",
			expected: "https://api.revenium.ai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeReveniumBaseURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "valid key",
			key:      "hak_1234567890",
			expected: true,
		},
		{
			name:     "invalid prefix",
			key:      "sk_1234567890",
			expected: false,
		},
		{
			name:     "too short",
			key:      "hak",
			expected: false,
		},
		{
			name:     "empty",
			key:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAPIKeyFormat(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithOptions(t *testing.T) {
	cfg := &Config{}

	// Test WithOpenAIAPIKey
	WithOpenAIAPIKey("test-key")(cfg)
	assert.Equal(t, "test-key", cfg.OpenAIAPIKey)

	// Test WithReveniumAPIKey
	WithReveniumAPIKey("hak_test")(cfg)
	assert.Equal(t, "hak_test", cfg.ReveniumAPIKey)

	// Test WithBaseURL
	WithBaseURL("https://custom.api.com")(cfg)
	assert.Equal(t, "https://custom.api.com", cfg.BaseURL)

	// Test WithAzureAPIKey
	WithAzureAPIKey("azure-key")(cfg)
	assert.Equal(t, "azure-key", cfg.AzureAPIKey)

	// Test WithAzureEndpoint
	WithAzureEndpoint("https://azure.openai.com")(cfg)
	assert.Equal(t, "https://azure.openai.com", cfg.AzureEndpoint)

	// Test WithAzureDisabled
	WithAzureDisabled(true)(cfg)
	assert.True(t, cfg.AzureDisabled)

	// Test WithDebug
	WithDebug(true)(cfg)
	assert.True(t, cfg.Debug)
}
