// Getting Started with Revenium Azure OpenAI Middleware
//
// This is the simplest example to verify your Azure OpenAI setup is working.

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

	fmt.Println("Testing Azure OpenAI with Revenium tracking...")

	// Simple chat completion
	ctx := context.Background()
	metadata := map[string]interface{}{
		"organizationId": "org-getting-started",
		"productId":      "product-getting-started",
		"taskType":       "text-generation",
	}
	ctx = revenium.WithUsageMetadata(ctx, metadata)

	// IMPORTANT: For Azure OpenAI, the Model parameter must be your DEPLOYMENT NAME
	// This is the name you gave to your deployment in Azure Portal
	params := openai.ChatCompletionNewParams{
		Model: "gpt-5-mini-2", // Replace with your Azure deployment name
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
