package main

import (
	"context"
	"fmt"
	"log"

	"github.com/openai/openai-go/v3"
	"github.com/revenium/revenium-middleware-openai-go/revenium"
)

func main() {
	// Initialize middleware
	if err := revenium.Initialize(); err != nil {
		log.Fatalf("Failed to initialize middleware: %v", err)
	}

	client, err := revenium.GetClient()
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}
	defer client.Close()

	// Create context with metadata
	ctx := context.Background()
	metadata := map[string]interface{}{
		"organizationId": "org-basic-demo",
		"productId":      "prod-openai-middleware",
	}
	ctx = revenium.WithUsageMetadata(ctx, metadata)

	// Make request
	params := openai.ChatCompletionNewParams{
		Model: "o4-mini",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Say hello in Spanish and explain why Spanish is a beautiful language in 2-3 sentences."),
		},
		MaxCompletionTokens: openai.Int(1000),
	}

	resp, err := client.Chat().Completions().New(ctx, params)
	if err != nil {
		log.Fatalf("Failed to create chat completion: %v", err)
	}

	// Display response
	if len(resp.Choices) > 0 {
		fmt.Printf("Assistant: %s\n", resp.Choices[0].Message.Content)
	}

	fmt.Println("\nUsage data sent to Revenium! Check your dashboard")
}
