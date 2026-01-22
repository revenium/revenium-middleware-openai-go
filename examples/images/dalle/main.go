package main

import (
	"context"
	"fmt"
	"log"
	"os"

	revenium "github.com/revenium/revenium-middleware-openai-go/revenium"

	"github.com/openai/openai-go/v3"
)

func main() {
	// Initialize Revenium middleware
	// Required env vars:
	// - OPENAI_API_KEY: Your OpenAI API key
	// - REVENIUM_METERING_API_KEY: Your Revenium metering API key (hak_...)
	// Optional:
	// - REVENIUM_METERING_BASE_URL: Override Revenium API URL (default: https://api.revenium.ai)
	

	if err := revenium.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Revenium: %v", err)
	}

	// Get the Revenium client
	client, err := revenium.GetClient()
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}
	defer client.Close()

	// Create context with optional metadata for billing/tracking
	ctx := revenium.WithUsageMetadata(context.Background(), map[string]interface{}{
		"organizationId": "org-123",
		"productId":      "prod-456",
		"taskType":       "image-generation",
		"agent":          "dalle-example",
	})

	// Generate an image with DALL-E
	fmt.Println("Generating image with DALL-E...")

	resp, err := client.Images().Generate(ctx, openai.ImageGenerateParams{
		Prompt: "A serene mountain landscape with a lake at sunset, digital art style",
		Model:  openai.ImageModelDallE3,
		Size:   openai.ImageGenerateParamsSize1024x1024,
		N:      openai.Int(1),
	})

	if err != nil {
		log.Fatalf("Image generation failed: %v", err)
	}

	// Print results
	fmt.Printf("\nGenerated %d image(s):\n", len(resp.Data))
	for i, img := range resp.Data {
		fmt.Printf("  [%d] URL: %s\n", i+1, img.URL)
		if img.RevisedPrompt != "" {
			fmt.Printf("      Revised Prompt: %s\n", img.RevisedPrompt)
		}
	}

	// Flush to ensure metering data is sent before exit
	client.Flush()

	fmt.Println("\n✅ Image generated and metering data sent to Revenium!")
	fmt.Println("Check your Revenium dashboard for the metered usage.")
}

// For DEV testing, set these environment variables:
// export OPENAI_API_KEY="sk-..."
// export REVENIUM_METERING_API_KEY="hak_..."

//
// Run with: go run main.go

func init() {
	// Validate required env vars
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}
	if os.Getenv("REVENIUM_METERING_API_KEY") == "" {
		log.Fatal("REVENIUM_METERING_API_KEY environment variable is required")
	}
}
