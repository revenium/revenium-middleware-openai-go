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

	// Check for audio file argument
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <audio_file.mp3>")
	}

	audioFile := os.Args[1]
	f, err := os.Open(audioFile)
	if err != nil {
		log.Fatalf("Failed to open audio file: %v", err)
	}
	defer f.Close()

	// Create context with optional metadata for billing/tracking
	ctx := revenium.WithUsageMetadata(context.Background(), map[string]interface{}{
		"organizationId": "org-123",
		"productId":      "prod-456",
		"taskType":       "transcription",
		"agent":          "whisper-example",
	})

	// Transcribe audio with Whisper
	fmt.Printf("Transcribing audio file: %s\n", audioFile)

	resp, err := client.Audio().Transcriptions().New(ctx, openai.AudioTranscriptionNewParams{
		File:  f,
		Model: openai.AudioModelWhisper1,
	})

	if err != nil {
		log.Fatalf("Transcription failed: %v", err)
	}

	// Flush to ensure metering data is sent before exit
	client.Flush()

	// Print results
	fmt.Println("\n✅ Transcription completed!")
	fmt.Printf("Response type: %T\n", resp)
	fmt.Println("Metering data sent to Revenium!")
	fmt.Println("Check your Revenium dashboard for the metered usage.")
}

// For DEV testing, set these environment variables:
// export OPENAI_API_KEY="sk-..."
// export REVENIUM_METERING_API_KEY="hak_..."

//
// Run with: go run main.go sample_audio.mp3

func init() {
	// Validate required env vars
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}
	if os.Getenv("REVENIUM_METERING_API_KEY") == "" {
		log.Fatal("REVENIUM_METERING_API_KEY environment variable is required")
	}
}
