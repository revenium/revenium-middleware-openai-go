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
		"organizationId": "org-azure-streaming",
		"productId":      "prod-azure-middleware",
		"taskType":       "creative-writing",
		"agent":          "azure-story-generator",
	}
	ctx = revenium.WithUsageMetadata(ctx, metadata)

	// Create streaming chat completion
	// IMPORTANT: Use your Azure deployment name
	params := openai.ChatCompletionNewParams{
		Model: "gpt-5-mini-2", // Replace with your Azure deployment name
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a creative poet."),
			openai.UserMessage("Write a short poem about cloud computing and Azure."),
		},
		MaxCompletionTokens: openai.Int(1000),
	}

	stream, err := client.Chat().Completions().NewStreaming(ctx, params)
	if err != nil {
		log.Fatalf("Failed to create streaming completion: %v", err)
	}

	// Process the stream
	fmt.Print("Assistant: ")
	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}

	if err := stream.Err(); err != nil {
		log.Fatalf("\nStreaming error: %v", err)
	}

	// Close the stream
	if err := stream.Close(); err != nil {
		log.Printf("\nWarning: Failed to close stream: %v", err)
	}

	fmt.Println()
	fmt.Println("\nUsage data sent to Revenium! Check your dashboard")
}
