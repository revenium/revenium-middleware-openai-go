package main

import (
	"context"
	"fmt"
	"log"

	"github.com/openai/openai-go/v3"
	"github.com/revenium/revenium-middleware-openai-go/revenium"
)

func main() {
	// Initialize the middleware
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
		"organizationId": "org-azure-demo",
		"productId":      "prod-azure-integration",
		"taskType":       "question-answering",
		"agent":          "azure-assistant",
	}
	ctx = revenium.WithUsageMetadata(ctx, metadata)

	// Create a chat completion
	// IMPORTANT: For Azure OpenAI, the Model parameter must be your DEPLOYMENT NAME
	// This is the name you gave to your deployment in Azure Portal, NOT the OpenAI model name
	params := openai.ChatCompletionNewParams{
		Model: "gpt-5-mini-2", // Replace with your Azure deployment name
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful assistant that explains cloud computing concepts."),
			openai.UserMessage("Explain the difference between Azure OpenAI and OpenAI native API in 2-3 sentences."),
		},
		MaxCompletionTokens: openai.Int(500),
	}

	resp, err := client.Chat().Completions().New(ctx, params)
	if err != nil {
		log.Fatalf("Failed to create chat completion: %v", err)
	}

	// Display the response
	if len(resp.Choices) > 0 {
		fmt.Printf("Assistant: %s\n", resp.Choices[0].Message.Content)
	}

	fmt.Println("\nUsage data sent to Revenium! Check your dashboard")
}
