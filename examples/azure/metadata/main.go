package main

import (
	"context"
	"fmt"
	"log"

	"github.com/openai/openai-go/v3"
	"github.com/revenium/revenium-middleware-openai-go/revenium"
)

func main() {
	ctx := context.Background()

	// Initialize Revenium middleware
	if err := revenium.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Revenium: %v", err)
	}

	// Get the Revenium client
	client, err := revenium.GetClient()
	if err != nil {
		log.Fatalf("Failed to get Revenium client: %v", err)
	}
	defer client.Close()

	fmt.Println("Testing Azure OpenAI with complete metadata tracking...")
	fmt.Println()

	// Create comprehensive metadata with all supported fields
	metadata := map[string]interface{}{
		// Organization and product tracking
		"organizationId": "org-acme-corp",
		"productId":      "product-azure-assistant",
		"subscriptionId": "sub-enterprise-tier",

		// Task and agent tracking
		"taskType": "enterprise-support",
		"agent":    "azure-support-bot-v2",

		// Tracing and correlation
		"traceId": "trace-azure-abc123",

		// Quality metrics (0.0-1.0 scale)
		"responseQualityScore": 0.98,

		// Subscriber information (complete object)
		"subscriber": map[string]interface{}{
			"id":    "user-john-doe-789",
			"email": "john.doe@example.com",
			"credential": map[string]interface{}{
				"name":  "Production API Key",
				"value": "pk-prod-xyz789",
			},
		},
	}

	// Add metadata to context
	ctx = revenium.WithUsageMetadata(ctx, metadata)

	// Generate content with metadata tracking
	// IMPORTANT: Use your Azure deployment name
	params := openai.ChatCompletionNewParams{
		Model: "gpt-5-mini-2", // Replace with your Azure deployment name
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful assistant."),
			openai.UserMessage("Explain the benefits of using Azure OpenAI Service for enterprise applications in one paragraph."),
		},
		MaxCompletionTokens: openai.Int(500),
	}

	resp, err := client.Chat().Completions().New(ctx, params)
	if err != nil {
		log.Fatalf("Error generating content: %v", err)
	}

	if len(resp.Choices) > 0 {
		fmt.Println("Response:", resp.Choices[0].Message.Content)
		fmt.Println()
	}

	// Display usage information
	fmt.Println("Usage:")
	fmt.Printf("  Total tokens: %d\n", resp.Usage.TotalTokens)
	fmt.Printf("  Prompt tokens: %d\n", resp.Usage.PromptTokens)
	fmt.Printf("  Completion tokens: %d\n", resp.Usage.CompletionTokens)
	fmt.Println()

	fmt.Println("Tracking successful! Check your Revenium dashboard for complete metadata.")
	fmt.Println()
	fmt.Println("All metadata fields sent to Revenium")
}
