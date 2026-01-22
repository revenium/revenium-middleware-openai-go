package main

import (
	"context"
	"fmt"
	"io"
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
		"taskType":       "text-to-speech",
		"agent":          "tts-example",
	})

	// Generate speech with OpenAI TTS
	fmt.Println("Generating speech with OpenAI TTS...")

	text := "Hello! This is a test of the OpenAI text-to-speech API with Revenium metering."

	resp, err := client.Audio().Speech().New(ctx, openai.AudioSpeechNewParams{
		Input:          text,
		Model:          openai.SpeechModelTTS1,
		Voice:          openai.AudioSpeechNewParamsVoiceAlloy,
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
	})

	if err != nil {
		log.Fatalf("Speech generation failed: %v", err)
	}
	defer resp.Body.Close()

	// Save the audio to a file
	outputFile := "output.mp3"
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	bytesWritten, err := io.Copy(f, resp.Body)
	if err != nil {
		log.Fatalf("Failed to write audio: %v", err)
	}

	// Flush to ensure metering data is sent before exit
	client.Flush()

	fmt.Printf("\n✅ Speech generated and saved to %s (%d bytes)\n", outputFile, bytesWritten)
	fmt.Printf("   Input text: %d characters\n", len(text))
	fmt.Println("   Metering data sent to Revenium!")
	fmt.Println("   Check your Revenium dashboard for the metered usage.")
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
