package revenium

import (
	"strings"
)

// Provider represents the AI provider being used
type Provider string

const (
	ProviderOpenAI Provider = "OPENAI"
	ProviderAzure  Provider = "AZURE"
)

// DetectProvider detects which provider is being used based on configuration
func DetectProvider(cfg *Config) Provider {
	if cfg == nil {
		return ProviderOpenAI
	}

	// If Azure is explicitly disabled, use OpenAI
	if cfg.AzureDisabled {
		Debug("Azure OpenAI is explicitly disabled, using OpenAI native API")
		return ProviderOpenAI
	}

	// Check if Azure credentials are configured
	if cfg.AzureAPIKey != "" && cfg.AzureEndpoint != "" {
		Debug("Azure OpenAI credentials detected, using Azure OpenAI")
		return ProviderAzure
	}

	// Check if base URL indicates Azure
	if cfg.BaseURL != "" && isAzureURL(cfg.BaseURL) {
		Debug("Azure OpenAI URL detected in base URL, using Azure OpenAI")
		return ProviderAzure
	}

	// Default to OpenAI
	Debug("No Azure configuration detected, using OpenAI native API")
	return ProviderOpenAI
}

// isAzureURL checks if a URL is an Azure OpenAI URL
func isAzureURL(url string) bool {
	url = strings.ToLower(url)
	return strings.Contains(url, "azure.com") ||
		strings.Contains(url, "openai.azure.com") ||
		strings.Contains(url, ".azure.") ||
		strings.Contains(url, "azureopenai")
}

// IsOpenAI returns true if the provider is OpenAI
func (p Provider) IsOpenAI() bool {
	return p == ProviderOpenAI
}

// IsAzure returns true if the provider is Azure OpenAI
func (p Provider) IsAzure() bool {
	return p == ProviderAzure
}

// String returns the string representation of the provider
func (p Provider) String() string {
	return string(p)
}

// ModelSource returns the model source for metering
// For both OpenAI and Azure, the model source is "OPENAI"
func (p Provider) ModelSource() string {
	return "OPENAI"
}
