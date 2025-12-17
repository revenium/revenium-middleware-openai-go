// Getting Started with Revenium OpenAI Middleware
//
// This is the simplest example to verify your setup is working.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/openai/openai-go/v3"
	"github.com/revenium/revenium-middleware-openai-go/revenium"
)

func main() {
	// Initialize middleware (automatically uses environment variables)
	if err := revenium.Initialize(); err != nil {
		log.Fatalf("Failed to initialize middleware: %v", err)
	}

	client, err := revenium.GetClient()
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}
	defer client.Close()

	fmt.Println("Testing OpenAI with Revenium tracking...")

	// Create context with usage metadata
	// All supported metadata fields shown below (uncomment as needed)
	ctx := context.Background()
	metadata := map[string]interface{}{
		// Required/Common fields
		"organizationId": "org-getting-started",
		"productId":      "product-getting-started",
		"taskType":       "text-generation",

		// Optional: Subscription and agent tracking
		// "subscriptionId": "sub-premium-tier",
		// "agent":          "my-agent-name",

		// Optional: Distributed tracing
		// "traceId": "trace-abc123-def456",

		// Optional: Quality scoring (0.0-1.0 scale)
		// "responseQualityScore": 0.95,

		// Optional: Subscriber details (for user attribution)
		// "subscriber": map[string]interface{}{
		// 	"id":    "user-123",
		// 	"email": "user@example.com",
		// 	"credential": map[string]interface{}{
		// 		"name":  "API Key Name",
		// 		"value": "key-identifier",
		// 	},
		// },
	}
	ctx = revenium.WithUsageMetadata(ctx, metadata)

	params := openai.ChatCompletionNewParams{
		Model: "gpt-4o",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Please verify you are ready to assist me."),
		},
		MaxCompletionTokens: openai.Int(100),
	}

	resp, err := client.Chat().Completions().New(ctx, params)
	if err != nil {
		log.Fatalf("Failed to create chat completion: %v", err)
	}

	if len(resp.Choices) > 0 {
		fmt.Println("Response:", resp.Choices[0].Message.Content)
	}

	fmt.Println("\nUsage:")
	fmt.Printf("  Total tokens: %d\n", resp.Usage.TotalTokens)
	fmt.Printf("  Prompt tokens: %d\n", resp.Usage.PromptTokens)
	fmt.Printf("  Completion tokens: %d\n", resp.Usage.CompletionTokens)

	fmt.Println("\nTracking successful! Check your Revenium dashboard.")
}
