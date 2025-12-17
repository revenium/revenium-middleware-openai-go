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

	fmt.Println("Testing OpenAI with complete metadata tracking...")
	fmt.Println()

	// Create comprehensive metadata with all supported fields
	metadata := map[string]interface{}{
		// Organization and product tracking
		"organizationId": "org-acme-corp",
		"productId":      "product-ai-assistant",
		"subscriptionId": "sub-premium-tier",

		// Task and agent tracking
		"taskType": "customer-support",
		"agent":    "support-bot-v2",

		// Tracing and correlation
		"traceId": "trace-abc123-def456",

		// Quality metrics (0.0-1.0 scale)
		"responseQualityScore": 0.95,

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
	params := openai.ChatCompletionNewParams{
		Model: "gpt-4o",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful assistant."),
			openai.UserMessage("Explain the importance of metadata in AI systems in one paragraph."),
		},
		MaxCompletionTokens: openai.Int(500),
		Temperature:         openai.Float(0.7), // This will be automatically captured
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
	fmt.Println("All metadata fields sent")
}
