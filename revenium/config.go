package revenium

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const (
	defaultReveniumBaseURL = "https://api.revenium.ai"
)

// Config holds all configuration for the Revenium middleware
type Config struct {
	// OpenAI API configuration
	OpenAIAPIKey string
	OpenAIOrgID  string
	BaseURL      string

	// Revenium metering configuration
	ReveniumAPIKey  string
	ReveniumBaseURL string

	// Azure OpenAI configuration
	AzureAPIKey     string
	AzureEndpoint   string
	AzureAPIVersion string
	AzureDisabled   bool

	// Debug configuration
	Debug bool
}

// Option is a functional option for configuring Config
type Option func(*Config)

// WithOpenAIAPIKey sets the OpenAI API key
func WithOpenAIAPIKey(key string) Option {
	return func(c *Config) {
		c.OpenAIAPIKey = key
	}
}

// WithOpenAIOrgID sets the OpenAI organization ID
func WithOpenAIOrgID(orgID string) Option {
	return func(c *Config) {
		c.OpenAIOrgID = orgID
	}
}

// WithBaseURL sets the base URL for OpenAI API
func WithBaseURL(url string) Option {
	return func(c *Config) {
		c.BaseURL = url
	}
}

// WithReveniumAPIKey sets the Revenium API key
func WithReveniumAPIKey(key string) Option {
	return func(c *Config) {
		c.ReveniumAPIKey = key
	}
}

// WithReveniumBaseURL sets the Revenium base URL
func WithReveniumBaseURL(url string) Option {
	return func(c *Config) {
		c.ReveniumBaseURL = url
	}
}

// WithAzureAPIKey sets the Azure OpenAI API key
func WithAzureAPIKey(key string) Option {
	return func(c *Config) {
		c.AzureAPIKey = key
	}
}

// WithAzureEndpoint sets the Azure OpenAI endpoint
func WithAzureEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.AzureEndpoint = endpoint
	}
}

// WithAzureAPIVersion sets the Azure OpenAI API version
func WithAzureAPIVersion(version string) Option {
	return func(c *Config) {
		c.AzureAPIVersion = version
	}
}

// WithAzureDisabled disables Azure OpenAI support
func WithAzureDisabled(disabled bool) Option {
	return func(c *Config) {
		c.AzureDisabled = disabled
	}
}

// WithDebug enables or disables debug logging programmatically
func WithDebug(debug bool) Option {
	return func(c *Config) {
		c.Debug = debug
	}
}

// loadFromEnv loads configuration from environment variables and .env files
func (c *Config) loadFromEnv() error {
	// First, try to load .env files automatically
	c.loadEnvFiles()

	// Then load from environment variables (which may have been set by .env files)
	c.OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	c.OpenAIOrgID = os.Getenv("OPENAI_ORG_ID")
	c.ReveniumAPIKey = os.Getenv("REVENIUM_METERING_API_KEY")
	baseURL := getEnvOrDefault("REVENIUM_METERING_BASE_URL", defaultReveniumBaseURL)
	c.ReveniumBaseURL = NormalizeReveniumBaseURL(baseURL)

	c.AzureAPIKey = os.Getenv("AZURE_OPENAI_API_KEY")
	c.AzureEndpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	c.AzureAPIVersion = os.Getenv("AZURE_OPENAI_API_VERSION")

	c.Debug = os.Getenv("REVENIUM_DEBUG") == "true"

	if os.Getenv("REVENIUM_AZURE_DISABLE") == "1" || os.Getenv("REVENIUM_AZURE_DISABLE") == "true" {
		c.AzureDisabled = true
	}

	SetGlobalDebug(c.Debug)
	Debug("Loading configuration from environment variables")

	return nil
}

// loadEnvFiles loads environment variables from .env files
func (c *Config) loadEnvFiles() {
	// Try to load .env files in order of preference
	envFiles := []string{
		".env.local", // Local overrides (highest priority)
		".env",       // Main env file
	}

	var loadedFiles []string

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Try current directory and parent directories
	searchDirs := []string{
		cwd,
		filepath.Dir(cwd),
		filepath.Join(cwd, ".."),
	}

	for _, dir := range searchDirs {
		for _, envFile := range envFiles {
			envPath := filepath.Join(dir, envFile)

			// Check if file exists
			if _, err := os.Stat(envPath); err == nil {
				// Try to load the file
				if err := godotenv.Load(envPath); err == nil {
					loadedFiles = append(loadedFiles, envPath)
				}
			}
		}
	}

	// Log loaded files (only if we have a logger initialized)
	if len(loadedFiles) > 0 {
		// We can't use Debug here because logger might not be initialized yet
		// So we'll just silently load the files
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ReveniumAPIKey == "" {
		return NewConfigError("REVENIUM_METERING_API_KEY is required", nil)
	}

	if !isValidAPIKeyFormat(c.ReveniumAPIKey) {
		return NewConfigError("invalid Revenium API key format", nil)
	}

	Debug("Configuration validation passed")
	return nil
}

// isValidAPIKeyFormat checks if the API key has a valid format
func isValidAPIKeyFormat(key string) bool {
	// Revenium API keys should start with "hak_"
	if len(key) < 4 {
		return false
	}
	return key[:4] == "hak_"
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// NormalizeReveniumBaseURL normalizes the base URL to a consistent format
// It handles various input formats and returns a normalized base URL without trailing slash
// The endpoint path (/meter/v2/ai/completions) is appended by sendMeteringRequest
func NormalizeReveniumBaseURL(baseURL string) string {
	if baseURL == "" {
		return defaultReveniumBaseURL
	}

	// Remove trailing slash if present
	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	// If it already ends with /meter/v2, remove /meter/v2 (legacy format)
	if len(baseURL) >= 9 && baseURL[len(baseURL)-9:] == "/meter/v2" {
		return baseURL[:len(baseURL)-9]
	}

	// If it ends with /meter, remove /meter (legacy format)
	if len(baseURL) >= 6 && baseURL[len(baseURL)-6:] == "/meter" {
		return baseURL[:len(baseURL)-6]
	}

	// If it ends with /v2, remove /v2 (legacy format)
	if len(baseURL) >= 3 && baseURL[len(baseURL)-3:] == "/v2" {
		return baseURL[:len(baseURL)-3]
	}

	// Return the base URL as-is (should be just the domain)
	return baseURL
}
